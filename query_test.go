package huelio

import "fmt"

func ExampleParseAction() {
	fmt.Printf("%#v\n", ParseAction("on"))
	fmt.Printf("%#v\n", ParseAction("off"))
	fmt.Printf("%#v\n", ParseAction("stuff"))
	// Output: huelio.Action{Scene:"", OnOff:"on"}
	// huelio.Action{Scene:"", OnOff:"off"}
	// huelio.Action{Scene:"stuff", OnOff:""}
}
