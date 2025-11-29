package data

import (
	"context"
	"fmt"
	"sync"
	"time"

	bestdoriapi "github.com/WindowsSov8forUs/bestdori-api-go"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/dto"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/post"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/songs"
	"github.com/WindowsSov8forUs/bestdori-api-go/uniapi"

	"anon-bestdori-database/config"
	"anon-bestdori-database/database"
	"anon-bestdori-database/files"
	"anon-bestdori-database/pkg/log"
)

type DataUpdater struct {
	bestdoriAPI   *uniapi.UniAPI
	niconiAPI     *uniapi.UniAPI
	db            *database.Database
	conf          *config.Config
	ctx           context.Context
	postGapLimit  int
	mu            sync.Mutex
	updateRunning bool
	updateDone    chan struct{}
}

var retryAttempts int

func setRetryAttempts(n int) {
	if n <= 0 {
		retryAttempts = 1
		return
	}
	retryAttempts = n
}

func NewDataUpdater(db *database.Database, conf *config.Config, ctx context.Context) *DataUpdater {
	bestdoriapi.RegisterLogger(log.GetLogger())

	bestdoriAPI := bestdoriapi.NewBestdoriAPI(conf.API.Proxy, conf.API.Timeout, conf.API.Retry)
	niconiAPI := bestdoriapi.NewNiconiAPI(conf.API.Proxy, conf.API.Timeout, conf.API.Retry)

	setRetryAttempts(conf.API.Retry)

	return &DataUpdater{
		bestdoriAPI:  bestdoriAPI,
		niconiAPI:    niconiAPI,
		db:           db,
		conf:         conf,
		ctx:          ctx,
		postGapLimit: conf.API.Gap,
	}
}

func retry(fn func() error) error {
	attempts := retryAttempts
	if attempts <= 0 {
		attempts = 5
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if isResponseStatusError(err) && i < attempts-1 {
				time.Sleep(3 * time.Second)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func isResponseStatusError(err error) bool {
	_, ok := err.(*uniapi.ResponseStatusError)
	return ok
}

func getSong(api *uniapi.UniAPI, id int) (*songs.Song, error) {
	var song *songs.Song
	err := retry(func() error {
		var err error
		song, err = songs.GetSong(api, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return song, nil
}

func downloadMusicJacket(jacket songs.Jacket) error {
	jacketName := "musicjacket/" + jacket.JacketImage + ".png"
	data, err := jacket.Bytes()
	if err != nil {
		return err
	}
	err = files.SaveAssets(jacketName, *data)
	if err != nil {
		return err
	}
	return nil
}

func downloadBGM(song *songs.Song) error {
	data, err := song.GetBGM()
	if err != nil {
		return err
	}
	bgmName := fmt.Sprintf("sound/bgm%03d.mp3", song.Id)
	err = files.SaveAssets(bgmName, *data)
	if err != nil {
		return err
	}
	return nil
}

func getChart(song *songs.Song, diff dto.ChartDifficultyName) (*dto.Chart, error) {
	var chart *dto.Chart
	err := retry(func() error {
		var err error
		chart, err = song.GetChart(diff)
		return err
	})
	if err != nil {
		return nil, err
	}
	return chart, nil
}

func getPost(bdAPI, nicoAPI *uniapi.UniAPI, id int) (*post.Post, error) {
	var p *post.Post
	err := retry(func() error {
		var err error
		p, err = post.GetPost(bdAPI, nicoAPI, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}
