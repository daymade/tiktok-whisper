package model

import (
	"tiktok-whisper/internal/downloader/model/misc"
	"time"
)

type PodcastDTO struct {
	Props        Props  `json:"props"`
	Page         string `json:"page"`
	Query        Query  `json:"query"`
	BuildID      string `json:"buildId"`
	AssetPrefix  string `json:"assetPrefix"`
	IsFallback   bool   `json:"isFallback"`
	Gsp          bool   `json:"gsp"`
	CustomServer bool   `json:"customServer"`
	ScriptLoader []any  `json:"scriptLoader"`
}

type Podcast struct {
	Type                     string             `json:"type"`
	Pid                      string             `json:"pid"`
	Title                    string             `json:"title"`
	Author                   string             `json:"author"`
	Brief                    string             `json:"brief"`
	Description              string             `json:"description"`
	SubscriptionCount        int                `json:"subscriptionCount"`
	Image                    misc.Image         `json:"image"`
	Color                    misc.Color         `json:"color"`
	SyncMode                 string             `json:"syncMode"`
	EpisodeCount             int                `json:"episodeCount"`
	LatestEpisodePubDate     time.Time          `json:"latestEpisodePubDate"`
	SubscriptionStatus       string             `json:"subscriptionStatus"`
	SubscriptionPush         bool               `json:"subscriptionPush"`
	SubscriptionPushPriority string             `json:"subscriptionPushPriority"`
	SubscriptionStar         bool               `json:"subscriptionStar"`
	Status                   string             `json:"status"`
	Permissions              []misc.Permission  `json:"permissions"`
	PayType                  string             `json:"payType"`
	PayEpisodeCount          int                `json:"payEpisodeCount"`
	Podcasters               []misc.Podcaster   `json:"podcasters"`
	ReadTrackInfo            misc.ReadTrackInfo `json:"readTrackInfo"`
	HasPopularEpisodes       bool               `json:"hasPopularEpisodes"`
	Contacts                 []misc.Contact     `json:"contacts"`
	Episodes                 []Episode          `json:"episodes"`
}
type PageProps struct {
	Podcast Podcast `json:"podcast"`
}
type Props struct {
	PageProps PageProps `json:"pageProps"`
	NSsg      bool      `json:"__N_SSG"`
}
type Query struct {
	ID string `json:"id"`
}
