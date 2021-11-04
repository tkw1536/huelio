package huelio

import "fmt"

func ExampleParseChange() {
	fmt.Printf("%#v\n", ParseChange(""))
	fmt.Printf("%#v\n", ParseChange("on"))
	fmt.Printf("%#v\n", ParseChange("off"))
	fmt.Printf("%#v\n", ParseChange("stuff"))
	// Output: huelio.Change{Scene:"", OnOff:""}
	// huelio.Change{Scene:"", OnOff:"on"}
	// huelio.Change{Scene:"", OnOff:"off"}
	// huelio.Change{Scene:"stuff", OnOff:""}
}
