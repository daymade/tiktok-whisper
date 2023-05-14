package misc

type Image struct {
	PicURL       string `json:"picUrl"`
	LargePicURL  string `json:"largePicUrl"`
	MiddlePicURL string `json:"middlePicUrl"`
	SmallPicURL  string `json:"smallPicUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type Podcaster struct {
	Type          string        `json:"type"`
	UID           string        `json:"uid"`
	Avatar        Avatar        `json:"avatar"`
	Nickname      string        `json:"nickname"`
	IsNicknameSet bool          `json:"isNicknameSet"`
	Bio           string        `json:"bio"`
	Gender        string        `json:"gender"`
	IsCancelled   bool          `json:"isCancelled"`
	ReadTrackInfo ReadTrackInfo `json:"readTrackInfo"`
	IPLoc         string        `json:"ipLoc"`
}

type Contact struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
	URL  string `json:"url,omitempty"`
}

type Color struct {
	Original string `json:"original"`
	Light    string `json:"light"`
	Dark     string `json:"dark"`
}

type Avatar struct {
	Picture Picture `json:"picture"`
}

type ReadTrackInfo struct {
}

type Enclosure struct {
	URL string `json:"url"`
}

type Source struct {
	Mode string `json:"mode"`
	URL  string `json:"url"`
}

type Media struct {
	ID       string `json:"id"`
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Source   Source `json:"source"`
}
type WechatShare struct {
	Style string `json:"style"`
}

type Permission struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Picture struct {
	PicURL       string `json:"picUrl"`
	LargePicURL  string `json:"largePicUrl"`
	MiddlePicURL string `json:"middlePicUrl"`
	SmallPicURL  string `json:"smallPicUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Format       string `json:"format"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}
