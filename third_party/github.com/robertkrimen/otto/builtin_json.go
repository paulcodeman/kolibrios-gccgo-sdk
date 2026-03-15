package otto

func builtinJSON_parse(call FunctionCall) Value {
	panic(call.runtime.panicTypeError("JSON.parse is not supported in this build"))
}

func builtinJSON_stringify(call FunctionCall) Value {
	panic(call.runtime.panicTypeError("JSON.stringify is not supported in this build"))
}
