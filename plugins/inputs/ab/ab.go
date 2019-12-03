package ab

import (	
	"fmt"
	"plctag"	
	"reflect"
	"sort"
	"strings"
	"strconv"
	"errors"
	"time"
		
	"github.com/influxdata/telegraf"	
	"github.com/influxdata/telegraf/plugins/inputs"
)

type CLX struct {
	Controller        string            `toml:"controller"`	
	Path              string            `toml:"path"`	
	Debug             int               `toml:"debug"`			
	Time_out          int               `toml:"time_out"`
	Discrete_Inputs   []tag             `toml:"discrete_inputs"`
	Coils             []tag             `toml:"coils"`
	Holding_Registers []tag             `toml:"holding_registers"`
	Input_Registers   []tag             `toml:"input_registers"`
	registers         []register	

	is_initialized    bool
	is_tagCreate      bool
}

type register struct {
	reg_type        string	
	tags            []tag
	plcTags         []plcTag
	values			map[string]interface{}	
}

type tag struct {
	Name       string    `toml:"name"`	
	Data_Type  string    `toml:"data_type"`	
	Address    string    `toml:"address"`	
}

type plcTag struct {	
	key			string
	name 		string
	tag_path 	string
	elem_count 	int
	data_type 	string
	elem_size   int
	fd          int32	
}

const (
	C_DISCRETE_INPUTS   = "Discrete_Inputs"
	C_COILS             = "Coils"
	C_HOLDING_REGISTERS = "Holding_Registers"
	C_INPUT_REGISTERS   = "Input_Registers"
)

var CLXConfig = ` 
 time_out = 5000
 debug    = 1
 controller = "192.168.0.200"
 path = "1,0"
 # TAG_PATH     = "protocol=ab_eip&gateway=192.168.0.250&path=1,0&cpu=LGX&elem_size=1&elem_count=1&debug=1&name=pcomm_test_bool" 
 ## Digital Variables, Discrete Inputs and Coils
 ## name    - the variable name
 ## address - variable address
 
 discrete_inputs = []
 coils = [] 
 
 ## Analog Variables, Input Registers and Holding Registers
 ## name       - the variable name 
 ## data_type  - BOOL, FLOAT, INT, DINT
 ## address    - variable address
 
 holding_registers = [
   { name = "PowerFactor", address = "Register[10]"},
   { name = "Voltage",     address = "Register[1]"},   
   { name = "Energy",      address = "Register[5]"},
   { name = "Current",     address = "Register[6]"},
   { name = "Frequency",   address = "Register_1"},
   { name = "Power",       address = "Register_2"},      
 ] 
 input_registers = []
`

func (s *CLX) SampleConfig() string {
	return CLXConfig
}

func (s *CLX) Description() string {
	return "Allen Bradley client"
}

func initialization(m *CLX) error {
	r := reflect.ValueOf(m).Elem()
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		
		if f.Type().String() == "[]ab.tag" {
			tags := f.Interface().([]tag)
			reg_type := r.Type().Field(i).Name

			if len(tags) == 0 {
				continue
			}
	
			var plc_tag_addrs = make(map[string]struct{
				address         []int
				address_range  []struct{
					start int
					end   int
				} 
				data_type      string
			})
						
			for _, tag := range tags {									
					key := tag.Address
					openBracket := strings.Index(tag.Address, "[")
					closeBracket := strings.Index(tag.Address, "]")
					length := 1

					if openBracket >= 0 && closeBracket >= 0  && closeBracket == len(tag.Address) - 1 {
						num := tag.Address[openBracket + 1:closeBracket]
						register_name := tag.Address[:openBracket]						
																		
						i1, err := strconv.Atoi(num)
						if err != nil {
							fmt.Println(i1)
						}
					
						length = i1		
						key = register_name								
					}
								
					reg_array := plc_tag_addrs[key].address		
					reg_array = append(reg_array, length)						
					sort.Slice(reg_array, func(i, j int) bool { return reg_array[i] < reg_array[j] })

					plc_tag_addrs[key] = struct {
						address         []int
						address_range  []struct{
							start int
							end   int
						} 
						data_type      string
					}{
						address : reg_array,
						data_type : tag.Data_Type,
					}																						
			}
							
			for key, _ := range plc_tag_addrs {
				var registers_range  []struct{
					start int
					end   int
				} 

				ii := 0
				addrs_t := plc_tag_addrs[key].address
				sort.Slice(addrs_t, func(i, j int) bool { return addrs_t[i] < addrs_t[j] })

				for range addrs_t {				
					if ii < len(addrs_t) {
						start := addrs_t[ii]
						end := start
	
						for ii < len(addrs_t)-1 && addrs_t[ii+1]-addrs_t[ii] == 1 {
							end = addrs_t[ii+1]
							ii++
						}
						ii++

						var register_range struct {
							start int
							end   int
						}			
						
						register_range.start = start
						register_range.end = end - start + 1

						registers_range = append(registers_range, register_range)							
					}
				}
				
				plc_tag_addrs[key] = struct {
					address         []int
					address_range   []struct{
						start int
						end   int
					} 
					data_type      string
				}{
					address : plc_tag_addrs[key].address,
					address_range : registers_range,
					data_type : plc_tag_addrs[key].data_type,
				}								
			}
															
			var plcTags []plcTag			
			
			for key, _ := range plc_tag_addrs {	
				for _, ar := range plc_tag_addrs[key].address_range {
					data_type := plc_tag_addrs[key].data_type
					name := ""
					elem_count := 0
					elem_size :=0 

					//fmt.Printf("range %v len %v \n", plc_tag_addrs[key].address_range, len(plc_tag_addrs[key].address_range))
					if plc_tag_addrs[key].address_range[0].end > 1 {
						name = key+"["+ strconv.Itoa(ar.start) +"]"
						elem_count = ar.end
					}else{
						name = key
						elem_count = 1
					}

					tag_path := "protocol=ab_eip"
					tag_path += "&gateway=" + m.Controller
					tag_path += "&path=" + m.Path
					tag_path += "&cpu=LGX"
			
					if data_type == "BOOL"  {
						elem_size = 1
					}else if data_type == "FLOAT" {
						elem_size = 4
						tag_path += "&elem_size=4"
					}else if data_type == "INT"  {	
						elem_size = 2
					}else if data_type == "DINT"  {	
						elem_size = 4
					}

					tag_path += "&elem_size="+strconv.Itoa(elem_size)
					tag_path += "&elem_count="+strconv.Itoa(elem_count)

					if m.Debug == 1 {
						tag_path += "&debug=1"
					}else{
						tag_path += "&debug=0"
					}

					tag_path += "&name="+name	
					//Println(tag_path)										

					plcTags =  append(plcTags, plcTag {
						key: key,
						name : name,
						tag_path: tag_path, 
						elem_count: elem_count,
						data_type : data_type,	
						elem_size: elem_size,					
					})
				}				
			} 
																				
			m.registers = append(m.registers, register{
				reg_type:reg_type, 
				tags:tags, 
				plcTags:plcTags,
				values: make(map[string]interface{}),
			})
		}
	}
	m.is_initialized = true

	return nil
}

func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] {
		} else {
			encountered[elements[v]] = true
			result = append(result, elements[v])
		}
	}

	return result
}

func (m *CLX) createTags() error {
	for i:=0 ; i < len(m.registers); i++ {	
		for j:=0 ; j < len(m.registers[i].plcTags); j++ {			
			if m.registers[i].plcTags[j].fd = plctag.Create(m.registers[i].plcTags[j].tag_path, m.Time_out); m.registers[i].plcTags[j].fd < 0 {				
				err := fmt.Sprintf("ERROR %s: Could not create tag!\n", plctag.DecodeError(int(m.registers[i].plcTags[j].fd)))
				return errors.New(err)				
			}
			
			if rc:= plctag.Status(m.registers[i].plcTags[j].fd); rc != plctag.STATUS_OK {
				err := fmt.Sprintf("ERROR %s: Error setting up tag internal state.\n", plctag.DecodeError(rc))
				return errors.New(err)
			}									
		}
	}

	m.is_tagCreate = true

	return nil
}

func (m *CLX) GetTags() error {
	for _, r := range m.registers {
		for _, pt := range r.plcTags {
			if rc:= plctag.Read(pt.fd, m.Time_out); rc != plctag.STATUS_OK {
				err := fmt.Sprintf("ERROR: Unable to read the data! Got error code %d: %s, fd:%v\n", rc, plctag.DecodeError(rc), pt.fd)
				return errors.New(err)
			}

			for i:=0; i < pt.elem_count ; i++ {
				var v interface{}									
				if pt.data_type == "BOOL"  {					
					v_bool := plctag.GetUint8(pt.fd, i*pt.elem_size)							
					if v_bool == 255 {
						v = 1
					}else{
						v = 0
					}					
				}else if pt.data_type == "FLOAT" {
					v = plctag.GetFloat32(pt.fd, i*pt.elem_size)
				}else if pt.data_type == "INT"  {	
					v = plctag.GetInt16(pt.fd, i*pt.elem_size)					
				}else if pt.data_type == "DINT"  {	
					v = plctag.GetInt32(pt.fd, i*pt.elem_size)					
				}	
				
				if pt.elem_count > 1{
					openBracket := strings.Index(pt.name, "[")
					closeBracket := strings.Index(pt.name, "]")

					
					num_str := pt.name[openBracket + 1:closeBracket]

					num, err := strconv.Atoi(num_str)
					if err != nil {
						return err
					}
					
					name := pt.key + "[" + strconv.Itoa(num + i) + "]"
					//fmt.Printf("%s - %v\n", name, v)
					r.values[name] = v
				}else{
					//fmt.Printf("%s - %v\n", pt.name, v)
					r.values[pt.name] = v
				}
			}

		}
	}

	for i:=0; i < len(m.registers); i++{
		for j:=0; j < len(m.registers[i].tags); j++{
			
		}
		
	}

	return nil
}

func addFields(r register) map[string]interface{} {
	fields := make(map[string]interface{})
	for _, t := range r.tags {		
		fields[t.Name] = r.values[t.Address]
	}	

	return fields
}

func (m *CLX) Gather(acc telegraf.Accumulator) error {
	start_global := time.Now()
	start := time.Now()

	fields := make(map[string]interface{})
	tags := make(map[string]string)
	
	if !m.is_initialized {
		err := initialization(m)
		if err != nil {
			return err
		}
	}
	time_init := int(time.Since(start) / time.Millisecond)
	
	start = time.Now()
	if !m.is_tagCreate {
		err := m.createTags()
		if err != nil{
			return err
		}
	}
	time_tag_create := int(time.Since(start) / time.Millisecond)
			
	start = time.Now()
	err := m.GetTags()
	if err != nil {		
		m.is_tagCreate = false
		return err
	}
	time_get_tags := int(time.Since(start) / time.Millisecond)

	start = time.Now()
	for _, reg := range m.registers {
		fields = addFields(reg)
		//fmt.Println(fields)
		acc.AddFields("clx."+reg.reg_type, fields, tags)
	}	
	time_add_fields := int(time.Since(start) / time.Millisecond)

	fields_time := make(map[string]interface{})
	tags_time := make(map[string]string)

	fields_time["01_init"]=time_init
	fields_time["02_tag_create"]=time_tag_create
	fields_time["03_get_tags"]=time_get_tags
	fields_time["04_add_fields"]=time_add_fields
	fields_time["05_start_global"]=int(time.Since(start_global) / time.Millisecond)

	acc.AddFields("clx.measure.time", fields_time, tags_time)
	
	return nil
}

func init() {
	inputs.Add("ab", func() telegraf.Input { return &CLX{} })
}
