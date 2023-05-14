package model

import (
	"tiktok-whisper/internal/downloader/model/misc"
	"time"
)

type Episode struct {
	Type           string             `json:"type"`
	Eid            string             `json:"eid"`
	Pid            string             `json:"pid"`
	Title          string             `json:"title"`
	Shownotes      string             `json:"shownotes"`
	Description    string             `json:"description"`
	Enclosure      misc.Enclosure     `json:"enclosure"`
	IsPrivateMedia bool               `json:"isPrivateMedia"`
	MediaKey       string             `json:"mediaKey"`
	Media          misc.Media         `json:"media"`
	ClapCount      int                `json:"clapCount"`
	CommentCount   int                `json:"commentCount"`
	PlayCount      int                `json:"playCount"`
	FavoriteCount  int                `json:"favoriteCount"`
	PubDate        time.Time          `json:"pubDate"`
	Status         string             `json:"status"`
	Duration       int                `json:"duration"`
	Podcast        Podcast            `json:"podcast"`
	Permissions    []misc.Permission  `json:"permissions"`
	PayType        string             `json:"payType"`
	WechatShare    misc.WechatShare   `json:"wechatShare"`
	ReadTrackInfo  misc.ReadTrackInfo `json:"readTrackInfo"`
	Labels         []any              `json:"labels"`
	Sponsors       []any              `json:"sponsors"`
	IsCustomized   bool               `json:"isCustomized"`
	IPLoc          string             `json:"ipLoc,omitempty"`
	Image          misc.Image         `json:"image,omitempty"`
}
