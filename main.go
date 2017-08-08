package critbit

//go:generate go get github.com/cheekybits/genny
//go:generate genny -in integer/map.go -out integerMaps.go gen "KeyType=int,int64,int32,int16,int8,uint,uintptr,uint64,uint32,uint16,uint8 ValueType=BUILTINS"
