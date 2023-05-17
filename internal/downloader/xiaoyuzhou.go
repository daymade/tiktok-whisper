package downloader

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"tiktok-whisper/internal/app/util/files"
	"tiktok-whisper/internal/downloader/model"
)

var pidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{24}$`)

func DownloadPodcast(input string, dir string) error {
	url, err := tryParsePodcastUrl(input)
	if err != nil {
		return err
	}

	podcastJson, err := getJsonScriptFromUrl(url)
	if err != nil {
		log.Fatal(fmt.Sprintf("get podcast data failed, %v", err))
	}

	var podcast model.PodcastDTO
	err = json.Unmarshal([]byte(podcastJson), &podcast)
	if err != nil {
		log.Fatal(fmt.Sprintf("deserialize podcast data failed, %v", err))
	}

	// $podcast_name is podcast.Props.PageProps.Podcast.Title
	podcastName := podcast.Props.PageProps.Podcast.Title

	// create dir for /data/xiaoyuzhou/$podcast_name
	podcastDir := filepath.Join(dir, validPath(podcastName))
	if _, err := os.Stat(podcastDir); os.IsNotExist(err) {
		err = os.MkdirAll(podcastDir, os.ModePerm)
		if err != nil {
			log.Fatal(fmt.Sprintf("failed to create directory %s: %v", podcastDir, err))
		}
	}

	// Get data for each episode
	for i, e := range podcast.Props.PageProps.Podcast.Episodes {
		// eid is the identifier of each episode
		log.Printf("Start downloading episode %d: %s", i+1, e.Title)
		err = DownloadEpisode(buildEpisodeUrl(e.Eid), podcastDir)
		if err != nil {
			log.Printf("Error downloading episode %d: %s - %v", i+1, e.Title, err)
			continue
		}
		log.Printf("Finished downloading episode %d: %s", i+1, e.Title)
	}
	return nil
}

func BatchDownloadEpisodes(urls []string, dir string) error {
	for _, url := range urls {
		err := DownloadEpisode(url, dir)
		if err != nil {
			log.Printf("Error downloading episode %s - %v", url, err)
			continue
		}
		log.Printf("Finished downloading episode %s", url)
	}
	return nil
}

func DownloadEpisode(url string, dir string) error {
	// Insert the URL of the episode you want to download.
	if isValidXiaoyuzhouEpisodeUrl(url) {
		audioUrl, episodeTitle, podcastName, err := getEpisodeInfo(url)
		if err != nil {
			return err
		}

		fileExtension := getAudioFileExtension(audioUrl)
		if fileExtension == "" {
			return fmt.Errorf("cannot get file extension for url %v", audioUrl)
		}

		podcastDir := buildPodcastDir(dir, podcastName)

		audioFilePath := fmt.Sprintf("%s/%s%s", podcastDir, validPath(episodeTitle), fileExtension)

		log.Printf("downloading episode %v into %v\n", episodeTitle, podcastDir)
		err = downloadFile(audioUrl, audioFilePath)
		if err != nil {
			return fmt.Errorf("downloadFile failed for url %v, err: %v", audioUrl, err)
		}

		return nil
	} else {
		return fmt.Errorf("url is not an valid episode, url: %v", url)
	}
}

func buildPodcastDir(dir string, podcastName string) string {
	absDir, err := files.GetAbsolutePath(dir)
	if err != nil {
		log.Fatalf("failed to get absolute path for %s: %v", dir, err)
	}

	lastPart := filepath.Base(absDir)
	podcastNameDir := validPath(podcastName)
	if lastPart != podcastNameDir {
		dir = filepath.Join(dir, podcastNameDir)
	}
	return dir
}

func validPath(path string) string {
	return strings.ReplaceAll(path, "/", "-")
}

// isValidXiaoyuzhouEpisodeUrl checks if the given URL is a valid xiaoyuzhou episode URL.
func isValidXiaoyuzhouEpisodeUrl(url string) bool {
	re := regexp.MustCompile(`^https?:\/\/(?:www\.)?xiaoyuzhoufm.com\/episode\/([0-9a-f]{24})`)
	return re.MatchString(url)
}

// getAudioFileExtension returns the file extension of the audio file URL.
func getAudioFileExtension(audioFileUrl string) string {
	supportFileExtensions := []string{".mp3", ".m4a", ".wav", ".ogg", ".flac", ".ape"}
	for _, ext := range supportFileExtensions {
		if strings.HasSuffix(audioFileUrl, ext) {
			return ext
		}
	}
	return ""
}

// getEpisodeInfo gets the audio URL and the title of the episode.
func getEpisodeInfo(url string) (audioUrl string, episodeTitle string, podcastName string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	// 获取音频文件的标题和下载地址
	audioTag := doc.Find(`meta[property="og:audio"]`).First()
	titleTag := doc.Find(`meta[property="og:title"]`).First()

	audioUrl, _ = audioTag.Attr("content")
	episodeTitle, _ = titleTag.Attr("content")

	if episodeTitle != "" && audioUrl != "" {
		doc.Find(".podcast-title").Each(func(i int, s *goquery.Selection) {
			podcastName = s.Text()
		})
		if podcastName == "" {
			podcastName = "未获取到播客名称"
		}

		return audioUrl, episodeTitle, podcastName, nil
	} else {
		return "", "", "", fmt.Errorf("cannot get audio url or episode title")
	}
}

// downloadFile downloads the file from the given URL and saves it to the local file system.
func downloadFile(url string, filepath string) error {
	absFilePath, err := files.GetAbsolutePath(filepath)
	if err != nil {
		return err
	}

	// Get the remote file's meta-information
	resp, err := http.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// BUG, remote md5 format is not valid, e.g. qB6aZ2qkNaHMu0PgSUeshA==
	remoteMD5 := resp.Header.Get("content-md5")
	remoteSize, err := strconv.ParseInt(resp.Header.Get("content-length"), 10, 64)
	if err != nil {
		return err
	}

	// Check if the local file exists
	fileInfo, err := os.Stat(absFilePath)
	if err == nil {
		// If the file sizes are different, need to download the file
		if fileInfo.Size() == remoteSize {
			// Calculate the local file's MD5
			localMD5, err := calculateFileMD5(absFilePath)
			if err != nil {
				return err
			}

			// If the MD5 values are the same, no need to download the file
			if localMD5 == remoteMD5 {
				log.Printf("local file %v is the same as remote file, no need to download\n", filepath)
				return nil
			}
		} else {
			log.Printf("local file size %v is different from remote file size %v, need to download the file\n",
				fileInfo.Size(), remoteSize)
		}
	}

	// The local file doesn't exist or is different from the remote file,
	// so we need to download the remote file
	resp, err = http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return files.WriteToFile(string(body), absFilePath)
}

func calculateFileMD5(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashInBytes), nil
}

func tryParsePodcastUrl(input string) (string, error) {
	// try to parse input as URL
	u, err := url.Parse(input)
	if err != nil {
		// PID is the identifier of podcast host
		// if it's not a valid URL, treat it as PID
		if pidRegexp.MatchString(input) {
			return buildPodcastUrl(input), nil
		} else {
			return "", fmt.Errorf("invalid PID or URL: %s", input)
		}
	}

	// if it's a valid URL, ensure it matches the required format
	if u.Host == "www.xiaoyuzhoufm.com" && strings.HasPrefix(u.Path, "/podcast/") {
		return u.String(), nil
	} else {
		return "", fmt.Errorf("URL does not match required format: https://www.xiaoyuzhoufm.com/podcast/$pid")
	}
}

func buildPodcastUrl(pid string) string {
	// https://www.xiaoyuzhoufm.com/podcast/61a9f093ca6141933d1a1c63
	return fmt.Sprintf("https://www.xiaoyuzhoufm.com/podcast/%s", pid)
}

func buildEpisodeUrl(eid string) string {
	// https://www.xiaoyuzhoufm.com/episode/64411602a79cc81470055c96
	return fmt.Sprintf("https://www.xiaoyuzhoufm.com/episode/%s", eid)
}

func getJsonScriptFromUrl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Get EID index in raw data
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		panic(err)
	}

	rawData, err := doc.Html()
	if err != nil {
		panic(err)
	}

	if strings.Contains(rawData, "找不到了") {
		panic("cannot found profile for url: " + url)
	}

	jsonData := doc.Find("#__NEXT_DATA__").Text()

	return jsonData, err
}
