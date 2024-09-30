package godatabase

import "fmt"

func buildDNS(connection Connection) string {
	var parseTime string
	if connection.ParseTime {
		parseTime = "True"
	} else {
		parseTime = "False"
	}

	var loc string
	if connection.Location == "" {
		loc = "Local"
	} else {
		loc = connection.Location
	}

	// build DNS string
	URI := "%v:%v@tcp(%v:%v)/%v?charset=%v&parseTime=%v&loc=%v"
	return fmt.Sprintf(URI,
		connection.Username,
		connection.Password,
		connection.Host,
		connection.Port,
		connection.Database,
		connection.Charset,
		parseTime,
		loc,
	)
}
