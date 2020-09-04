package main

import(
	// "bufio"
	"net/http"
	// "io"
	"io/ioutil"
	"fmt"
	"time"
	"os"
	"github.com/anaskhan96/soup"
	"github.com/jfbus/httprs"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/gdamore/tcell"
	"github.com/faiface/beep/effects"
	"unicode"
	"regexp"
	"encoding/json"
	"errors"
	"strconv"
)
func drawTextLine(screen tcell.Screen, x, y int, s string, style tcell.Style) {
	for _, r := range s {
		screen.SetContent(x, y, r, nil, style)
		x++
	}
}

type audioPanel struct {
	sampleRate beep.SampleRate
	streamer   beep.StreamSeeker
	ctrl       *beep.Ctrl
	resampler  *beep.Resampler
	volume     *effects.Volume
}

func newAudioPanel(sampleRate beep.SampleRate, streamer beep.StreamSeeker) *audioPanel {
	ctrl := &beep.Ctrl{Streamer: beep.Loop(1, streamer)}
	resampler := beep.ResampleRatio(4, 1, ctrl)
	volume := &effects.Volume{Streamer: resampler, Base: 2}
	return &audioPanel{sampleRate, streamer, ctrl, resampler, volume}
}

func (ap *audioPanel) play() {
	speaker.Play(ap.volume)
}

func (ap *audioPanel) draw(screen tcell.Screen) {
	mainStyle := tcell.StyleDefault.
		Background(tcell.NewHexColor(0x473437)).
		Foreground(tcell.NewHexColor(0xD7D8A2))
	statusStyle := mainStyle.
		Foreground(tcell.NewHexColor(0xDDC074)).
		Bold(true)

	screen.Fill(' ', mainStyle)

	drawTextLine(screen, 0, 0, "Welcome to the Speedy Player!", mainStyle)
	drawTextLine(screen, 0, 1, "Press [ESC] to quit.", mainStyle)
	drawTextLine(screen, 0, 2, "Press [SPACE] to pause/resume.", mainStyle)
	drawTextLine(screen, 0, 3, "Use keys in (?/?) to turn the buttons.", mainStyle)

	speaker.Lock()
	position := ap.sampleRate.D(ap.streamer.Position())
	length := ap.sampleRate.D(ap.streamer.Len())
	volume := ap.volume.Volume
	speed := ap.resampler.Ratio()
	speaker.Unlock()

	positionStatus := fmt.Sprintf("%v / %v", position.Round(time.Second), length.Round(time.Second))
	volumeStatus := fmt.Sprintf("%.1f", volume)
	speedStatus := fmt.Sprintf("%.3fx", speed)

	drawTextLine(screen, 0, 5, "Position (Q/W):", mainStyle)
	drawTextLine(screen, 16, 5, positionStatus, statusStyle)

	drawTextLine(screen, 0, 6, "Volume   (A/S):", mainStyle)
	drawTextLine(screen, 16, 6, volumeStatus, statusStyle)

	drawTextLine(screen, 0, 7, "Speed    (Z/X):", mainStyle)
	drawTextLine(screen, 16, 7, speedStatus, statusStyle)
}

func (ap *audioPanel) handle(event tcell.Event) (changed, quit bool) {
	switch event := event.(type) {
	case *tcell.EventKey:
		if event.Key() == tcell.KeyESC {
			return false, true
		}

		if event.Key() != tcell.KeyRune {
			return false, false
		}

		switch unicode.ToLower(event.Rune()) {
		case ' ':
			speaker.Lock()
			ap.ctrl.Paused = !ap.ctrl.Paused
			speaker.Unlock()
			return false, false

		case 'q', 'w':
			speaker.Lock()
			newPos := ap.streamer.Position()
			if event.Rune() == 'q' {
				newPos -= ap.sampleRate.N(time.Second)
			}
			if event.Rune() == 'w' {
				newPos += ap.sampleRate.N(time.Second)
			}
			if newPos < 0 {
				newPos = 0
			}
			if newPos >= ap.streamer.Len() {
				newPos = ap.streamer.Len() - 1
			}
			if err := ap.streamer.Seek(newPos); err != nil {
				report(err)
			}
			speaker.Unlock()
			return true, false

		case 'a':
			speaker.Lock()
			ap.volume.Volume -= 0.1
			speaker.Unlock()
			return true, false

		case 's':
			speaker.Lock()
			ap.volume.Volume += 0.1
			speaker.Unlock()
			return true, false

		case 'z':
			speaker.Lock()
			ap.resampler.SetRatio(ap.resampler.Ratio() * 15 / 16)
			speaker.Unlock()
			return true, false

		case 'x':
			speaker.Lock()
			ap.resampler.SetRatio(ap.resampler.Ratio() * 16 / 15)
			speaker.Unlock()
			return true, false
		}
	}
	return false, false
}
func report(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type PlayList struct {
	Duration     int         `json:"duration"`
	PermalinkURL string      `json:"permalink_url"`
	RepostsCount int         `json:"reposts_count"`
	Genre        string      `json:"genre"`
	Permalink    string      `json:"permalink"`
	PurchaseURL  interface{} `json:"purchase_url"`
	Description  interface{} `json:"description"`
	URI          string      `json:"uri"`
	LabelName    interface{} `json:"label_name"`
	TagList      string      `json:"tag_list"`
	SetType      string      `json:"set_type"`
	Public       bool        `json:"public"`
	TrackCount   int         `json:"track_count"`
	UserID       int         `json:"user_id"`
	LastModified time.Time   `json:"last_modified"`
	License      string      `json:"license"`
	Tracks       []struct {
		CommentCount int       `json:"comment_count,omitempty"`
		FullDuration int       `json:"full_duration,omitempty"`
		Downloadable bool      `json:"downloadable,omitempty"`
		CreatedAt    time.Time `json:"created_at,omitempty"`
		Description  string    `json:"description,omitempty"`
		Media        struct {
			Transcodings []interface{} `json:"transcodings"`
		} `json:"media,omitempty"`
		Title             string `json:"title,omitempty"`
		PublisherMetadata struct {
		} `json:"publisher_metadata,omitempty"`
		Duration          int         `json:"duration,omitempty"`
		HasDownloadsLeft  bool        `json:"has_downloads_left,omitempty"`
		ArtworkURL        interface{} `json:"artwork_url,omitempty"`
		Public            bool        `json:"public,omitempty"`
		Streamable        bool        `json:"streamable,omitempty"`
		TagList           string      `json:"tag_list,omitempty"`
		Genre             string      `json:"genre,omitempty"`
		ID                int         `json:"id,omitempty"`
		RepostsCount      int         `json:"reposts_count,omitempty"`
		State             string      `json:"state,omitempty"`
		LabelName         interface{} `json:"label_name,omitempty"`
		LastModified      time.Time   `json:"last_modified,omitempty"`
		Commentable       bool        `json:"commentable,omitempty"`
		Policy            string      `json:"policy,omitempty"`
		Visuals           interface{} `json:"visuals,omitempty"`
		Kind              string      `json:"kind,omitempty"`
		PurchaseURL       interface{} `json:"purchase_url,omitempty"`
		Sharing           string      `json:"sharing,omitempty"`
		URI               string      `json:"uri,omitempty"`
		SecretToken       interface{} `json:"secret_token,omitempty"`
		DownloadCount     int         `json:"download_count,omitempty"`
		LikesCount        int         `json:"likes_count,omitempty"`
		Urn               string      `json:"urn,omitempty"`
		License           string      `json:"license,omitempty"`
		PurchaseTitle     interface{} `json:"purchase_title,omitempty"`
		DisplayDate       time.Time   `json:"display_date,omitempty"`
		EmbeddableBy      string      `json:"embeddable_by,omitempty"`
		ReleaseDate       interface{} `json:"release_date,omitempty"`
		UserID            int         `json:"user_id,omitempty"`
		MonetizationModel string      `json:"monetization_model,omitempty"`
		WaveformURL       string      `json:"waveform_url,omitempty"`
		Permalink         string      `json:"permalink,omitempty"`
		PermalinkURL      string      `json:"permalink_url,omitempty"`
		User              struct {
			AvatarURL    string    `json:"avatar_url"`
			FirstName    string    `json:"first_name"`
			FullName     string    `json:"full_name"`
			ID           int       `json:"id"`
			Kind         string    `json:"kind"`
			LastModified time.Time `json:"last_modified"`
			LastName     string    `json:"last_name"`
			Permalink    string    `json:"permalink"`
			PermalinkURL string    `json:"permalink_url"`
			URI          string    `json:"uri"`
			Urn          string    `json:"urn"`
			Username     string    `json:"username"`
			Verified     bool      `json:"verified"`
			City         string    `json:"city"`
			CountryCode  string    `json:"country_code"`
		} `json:"user,omitempty"`
		PlaybackCount int `json:"playback_count,omitempty"`
	} `json:"tracks"`
	ID             int         `json:"id"`
	ReleaseDate    interface{} `json:"release_date"`
	DisplayDate    time.Time   `json:"display_date"`
	Sharing        string      `json:"sharing"`
	SecretToken    interface{} `json:"secret_token"`
	CreatedAt      time.Time   `json:"created_at"`
	LikesCount     int         `json:"likes_count"`
	Kind           string      `json:"kind"`
	Title          string      `json:"title"`
	PurchaseTitle  interface{} `json:"purchase_title"`
	ManagedByFeeds bool        `json:"managed_by_feeds"`
	ArtworkURL     string      `json:"artwork_url"`
	IsAlbum        bool        `json:"is_album"`
	User           struct {
		AvatarURL    string      `json:"avatar_url"`
		FirstName    string      `json:"first_name"`
		FullName     string      `json:"full_name"`
		ID           int         `json:"id"`
		Kind         string      `json:"kind"`
		LastModified time.Time   `json:"last_modified"`
		LastName     string      `json:"last_name"`
		Permalink    string      `json:"permalink"`
		PermalinkURL string      `json:"permalink_url"`
		URI          string      `json:"uri"`
		Urn          string      `json:"urn"`
		Username     string      `json:"username"`
		Verified     bool        `json:"verified"`
		City         interface{} `json:"city"`
		CountryCode  interface{} `json:"country_code"`
	} `json:"user"`
	PublishedAt  time.Time `json:"published_at"`
	EmbeddableBy string    `json:"embeddable_by"`
}
type Track struct {
	CommentCount int       `json:"comment_count"`
	FullDuration int       `json:"full_duration"`
	Downloadable bool      `json:"downloadable"`
	CreatedAt    time.Time `json:"created_at"`
	Description  string    `json:"description"`
	Media        struct {
		Transcodings []struct {
			URL      string `json:"url"`
			Preset   string `json:"preset"`
			Duration int    `json:"duration"`
			Snipped  bool   `json:"snipped"`
			Format   struct {
				Protocol string `json:"protocol"`
				MimeType string `json:"mime_type"`
			} `json:"format"`
			Quality string `json:"quality"`
		} `json:"transcodings"`
	} `json:"media"`
	Title             string `json:"title"`
	PublisherMetadata struct {
		Urn            string `json:"urn"`
		ContainsMusic  bool   `json:"contains_music"`
		Artist         string `json:"artist"`
		WriterComposer string `json:"writer_composer"`
		Publisher      string `json:"publisher"`
		Isrc           string `json:"isrc"`
		ID             int    `json:"id"`
	} `json:"publisher_metadata"`
	Duration          int         `json:"duration"`
	HasDownloadsLeft  bool        `json:"has_downloads_left"`
	ArtworkURL        interface{} `json:"artwork_url"`
	Public            bool        `json:"public"`
	Streamable        bool        `json:"streamable"`
	TagList           string      `json:"tag_list"`
	Genre             string      `json:"genre"`
	ID                int         `json:"id"`
	RepostsCount      int         `json:"reposts_count"`
	State             string      `json:"state"`
	LabelName         interface{} `json:"label_name"`
	LastModified      time.Time   `json:"last_modified"`
	Commentable       bool        `json:"commentable"`
	Policy            string      `json:"policy"`
	Visuals           interface{} `json:"visuals"`
	Kind              string      `json:"kind"`
	PurchaseURL       interface{} `json:"purchase_url"`
	Sharing           string      `json:"sharing"`
	URI               string      `json:"uri"`
	SecretToken       interface{} `json:"secret_token"`
	DownloadCount     int         `json:"download_count"`
	LikesCount        int         `json:"likes_count"`
	Urn               string      `json:"urn"`
	License           string      `json:"license"`
	PurchaseTitle     interface{} `json:"purchase_title"`
	DisplayDate       time.Time   `json:"display_date"`
	EmbeddableBy      string      `json:"embeddable_by"`
	ReleaseDate       interface{} `json:"release_date"`
	UserID            int         `json:"user_id"`
	MonetizationModel string      `json:"monetization_model"`
	WaveformURL       string      `json:"waveform_url"`
	Permalink         string      `json:"permalink"`
	PermalinkURL      string      `json:"permalink_url"`
	User              struct {
		AvatarURL    string    `json:"avatar_url"`
		FirstName    string    `json:"first_name"`
		FullName     string    `json:"full_name"`
		ID           int       `json:"id"`
		Kind         string    `json:"kind"`
		LastModified time.Time `json:"last_modified"`
		LastName     string    `json:"last_name"`
		Permalink    string    `json:"permalink"`
		PermalinkURL string    `json:"permalink_url"`
		URI          string    `json:"uri"`
		Urn          string    `json:"urn"`
		Username     string    `json:"username"`
		Verified     bool      `json:"verified"`
		City         string    `json:"city"`
		CountryCode  string    `json:"country_code"`
	} `json:"user"`
	PlaybackCount int `json:"playback_count"`
}
type Stream struct {
	URL string `json:"url"`
}
func getClientID() (id string, err error) {
	respSoup, err := soup.Get("https://soundcloud.com/discover")
	if err != nil {
		return "null" ,  errors.New("Error 01")
	}
	doc := soup.HTMLParse(respSoup)
	scripts := doc.FindAll("script")
	var link string
	for i, script := range scripts {
		if (i == (len(scripts) - 2)) {
			link = script.Attrs()["src"]
		}
	}
	// fmt.Println(link)
	resp, err := http.Get(link)
	if err != nil {
		return "null" ,  errors.New("Error 02")
	}
	body,	err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "null" ,  errors.New("Error 03")
	}
	// fmt.Println(string(body))
	r,	err := regexp.Compile("client_id")
	if err != nil {
		return "null" ,  errors.New("Error 04")
	}
	k:= r.FindStringIndex(string(body))[1]
	// fmt.Println(k)
	s:= string(body)[k + 2: k + 34]
	return s, nil
}

func Speaker(streamer beep.StreamSeekCloser, format beep.Format) {
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/30))

	screen, err := tcell.NewScreen()
	if err != nil {
		report(err)
	}
	err = screen.Init()
	if err != nil {
		report(err)
	}
	defer screen.Fini()

	ap := newAudioPanel(format.SampleRate, streamer)
 
	
	screen.Clear()
	ap.draw(screen)
	screen.Show()

	ap.play()

	seconds := time.Tick(time.Second)
	events := make(chan tcell.Event)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

loop:
	for {
		select {
		case event := <-events:
			changed, quit := ap.handle(event)
			if quit {
				break loop
			}
			if changed {
				screen.Clear()
				ap.draw(screen)
				screen.Show()
			}
		case <-seconds:
			screen.Clear()
			ap.draw(screen)
			screen.Show()
		}
	}
}

func main() {
	var urlPlayList string
	var num int

	fmt.Printf("Please Wait...\n")

	clientID, err :=getClientID()
	if err != nil {
		return
	}

	fmt.Printf("URL Playlist: ")
	fmt.Scanf("%s\n", &urlPlayList) 
	fmt.Printf("Please Wait...\n")

	// urlPlayList := "https://soundcloud.com/mnhatbk20/sets/yaaayaaayaaa"
	url := "https://api-v2.soundcloud.com/resolve?url="
	url = url + urlPlayList + "&client_id="+ clientID
	resp, _ := http.Get(url)
	body, _ := ioutil.ReadAll(resp.Body)
	var playlist PlayList
	json.Unmarshal([]byte(string(body)), &playlist)
	var trackIDs []int
	for _, track := range playlist.Tracks {
		trackIDs =  append(trackIDs, track.ID)	
	}

	var mp3Tracks []string
	var trackNames []string
	var mp3URLs []string
	for _, trackID := range trackIDs {
		url = "https://api-v2.soundcloud.com/tracks/"	
		url =  url + strconv.Itoa(trackID) + "?client_id=" + clientID
		resp, _ = http.Get(url)
		body, _ = ioutil.ReadAll(resp.Body)
		var track Track
		json.Unmarshal([]byte(string(body)), &track)
		// fmt.Println(track.Media.Transcodings[1].URL)
		mp3Tracks = append(mp3Tracks, track.Media.Transcodings[1].URL )
		trackNames =  append(trackNames, track.Title)	
		mp3URLs = append(mp3URLs, "" )
	}
	for i, trackName := range trackNames {
		fmt.Printf("%d: %s\n", i+1, trackName)
	}
	fmt.Printf("Please select a song (Number): ")
	fmt.Scanf("%d\n", &num) 
	fmt.Printf("Please Wait...\n")
	if (mp3URLs[num-1] == ""){
		url = mp3Tracks[num-1]	+ "?client_id=" + clientID
		resp, _ = http.Get(url)
		body, _ = ioutil.ReadAll(resp.Body)
		var stream Stream
		json.Unmarshal([]byte(string(body)), &stream)
		mp3URLs[num-1] = stream.URL
	}

	resp,_ = http.Get(mp3URLs[num-1])
	rs := httprs.NewHttpReadSeeker(resp)
	defer rs.Close()

	streamer, format, err := mp3.Decode(rs)
	if err != nil {
		return
	}
	defer streamer.Close()

	Speaker(streamer,format)

	
}




