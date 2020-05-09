package controllers

type StaticConfiguration struct {
	MemoryRequestMi int64 `default:"32" split_words:"true"`
	MemoryLimitMi   int64 `default:"128" split_words:"true"`
	CpuRequestMilli int64 `default:"100" split_words:"true"`
	CpuLimitMilli   int64 `default:"500" split_words:"true"`
	CpuUtilization  int32 `default:"400" split_words:"true"`
}
