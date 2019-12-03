package ab

import (
	"fmt"
	"os"
	"plctag"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type Ab struct {	
	Tag       string
}

const (
	TAG_PATH = "protocol=ab_eip&gateway=192.168.0.250&path=1,0&cpu=LGX&elem_size=4&elem_count=250&debug=1&name=TeRegister1"
	DATA_TIMEOUT = 5000
	ELEM_COUNT = 250
	ELEM_SIZE = 4
)


var AbConfig = `
  ## Set the amplitude
  tag = "nada"
`

func (s *Ab) SampleConfig() string {
	return AbConfig
}

func (s *Ab) Description() string {
	return "Inserts sine and cosine waves for demonstration purposes"
}

func (s *Ab) Gather(acc telegraf.Accumulator) error {	
	tag := plctag.Create(TAG_PATH, DATA_TIMEOUT);
	if (tag < 0) {
		tag := int(tag) // XXX: plctag.Create returns int32 for tags but DecodeError,Exit expects int
		fmt.Printf("ERROR %s: Could not create tag!\n", plctag.DecodeError(tag))
		os.Exit(tag)
	}

	var status int
	for {
		status = plctag.Status(tag)
		if status != plctag.STATUS_PENDING {
			break
		}
		time.Sleep(100)
	}
	if status != plctag.STATUS_OK {
		fmt.Printf("Error setting up tag internal state. Error %s\n", plctag.DecodeError(status))
		os.Exit(status)
	}

	result := plctag.Read(tag, DATA_TIMEOUT)
	if result != plctag.STATUS_OK {
		fmt.Printf("ERROR: Unable to read the data! Got error code %d: %s\n", result, plctag.DecodeError(result))
		os.Exit(result)
	}


	fields := make(map[string]interface{})
	
	for index := 0; index < ELEM_COUNT; index++ {
		//fmt.Printf("data[%d]=%d \n", index, plctag.GetInt32(tag, (index * ELEM_SIZE)))		
		//fields["ramp"] = plctag.GetInt32(tag, (index * ELEM_SIZE))

		if index == 0 {
			fields["ramp"] = plctag.GetInt32(tag, (index * ELEM_SIZE))
		}

		if index == 1 {
			fields["ramp2"] = plctag.GetInt32(tag, (index * ELEM_SIZE))
		}

		if index == 11 {
			fields["CV"] = plctag.GetInt32(tag, (index * ELEM_SIZE))
		}

		if index == 100 {
			fields["PV"] = plctag.GetInt32(tag, (index * ELEM_SIZE))
		}

		if index == 101 {
			fields["SP"] = plctag.GetInt32(tag, (index * ELEM_SIZE))
		}
	}
	
	tags := make(map[string]string)
	
	acc.AddFields("tag", fields, tags)

	return nil
}

func init() {
	inputs.Add("ab", func() telegraf.Input { return &Ab{} })
}
