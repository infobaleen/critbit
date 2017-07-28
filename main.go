package critbit

//go:generate genny -in templates/numbers.go -out generated.go gen "KeyType=int,int64,int32,int16,int8,uint,uint64,uint32,uint16,uint8 ValueType=BUILTINS"
