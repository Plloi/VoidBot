package main

type ChannelSettings struct {
	Active  bool  `json:"active"`
	Seconds int64 `json:"seconds"`
}

func NewChannelSettings() ChannelSettings {
	return ChannelSettings{
		Active:  false,
		Seconds: 5,
	}
}
