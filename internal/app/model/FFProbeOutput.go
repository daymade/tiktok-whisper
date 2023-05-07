package model

type FFProbeOutput struct {
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		SampleRate int    `json:"sample_rate,string"`
	} `json:"streams"`
}
