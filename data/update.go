package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/dto"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/songs"

	"anon-bestdori-database/files"
	"anon-bestdori-database/pkg/log"
)

type PostUpdateInfo struct {
	LastID int       `json:"last_id"`
	Time   time.Time `json:"time"`
}

func needsSongUpdate(existing *dto.SongInfo, newInfo dto.SongsAll8Info) bool {
	if existing == nil {
		return true
	}

	newJSON, _ := json.Marshal(newInfo)

	all8 := existing.SongsAll8Info
	oldJSON, _ := json.Marshal(all8)

	return !bytes.Equal(newJSON, oldJSON)
}

func (du *DataUpdater) Update() error {
	if err := du.ctx.Err(); err != nil {
		return err
	}
	if err := du.updateSongs(); err != nil {
		log.Errorf("failed to update songs data: %v", err)
		return err
	}
	if err := du.updatePosts(); err != nil {
		log.Errorf("failed to update posts data: %v", err)
		return err
	}
	return nil
}

func (du *DataUpdater) updateSongs() error {
	if err := du.ctx.Err(); err != nil {
		return err
	}
	all8, err := songs.GetAll8(du.bestdoriAPI)
	if err != nil {
		log.Errorf("failed to get songs all.8.json: %v", err)
		return err
	}

	idList := make([]int, 0, len(*all8))
	for idStr := range *all8 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		idList = append(idList, id)
	}
	slices.Sort(idList)

	for _, id := range idList {
		if err := du.ctx.Err(); err != nil {
			return err
		}
		info := (*all8)[strconv.Itoa(id)]
		existing, _ := du.db.GetSongByID(du.ctx, id)
		if !needsSongUpdate(existing, info) {
			continue
		}
		if _, err := du.UpdateSongByID(id); err != nil {
			log.Errorf("failed to update song %d: %v", id, err)
		}
	}
	return nil
}

func (du *DataUpdater) UpdateSongByID(id int) (bool, error) {
	if err := du.ctx.Err(); err != nil {
		return false, err
	}
	log.Infof("updating song %d info...", id)

	song, err := getSong(du.bestdoriAPI, id)
	if err != nil {
		return false, err
	}

	if err := du.db.UpsertSong(du.ctx, id, song.Info); err != nil {
		log.Errorf("failed to upsert song %d: %v", id, err)
		return true, err
	}
	log.Infof("updated song %d", id)

	jacketList := song.GetJacket()
	for _, jacket := range jacketList {
		jacketName := fmt.Sprintf("musicjacket/%s.png", jacket.JacketImage)
		if _, err := files.GetAssets(jacketName); err != nil {
			log.Infof("downloading missing jacket %s for song %d", jacket.JacketImage, id)
			if err := retry(func() error {
				return downloadMusicJacket(jacket)
			}); err != nil {
				log.Errorf("failed to update jacket %s for song %d: %v", jacket.JacketImage, id, err)
			} else {
				log.Infof("updated jacket %s for song %d", jacket.JacketImage, id)
			}
		}
	}

	bgmName := fmt.Sprintf("sound/bgm%03d.mp3", song.Id)
	if _, err := files.GetAssets(bgmName); err != nil {
		log.Infof("downloading missing BGM for song %d", id)
		if err := retry(func() error {
			return downloadBGM(song)
		}); err != nil {
			log.Errorf("failed to update BGM for song %d: %v", id, err)
		} else {
			log.Infof("updated BGM for song %d", id)
		}
	}

	diffs := []string{"easy", "normal", "hard", "expert", "special"}
	for _, diffStr := range diffs {
		chartID := fmt.Sprintf("%d-%s", id, diffStr)
		if existingChart, _ := du.db.GetChartByID(du.ctx, chartID); existingChart == nil {
			log.Infof("updating missing chart %s for song %d", chartID, id)
			diff := dto.ChartDifficultyName(diffStr)
			chart, err := getChart(song, diff)
			if err != nil {
				if _, ok := err.(*bestdori.NotExistError); !ok {
					log.Errorf("failed to get chart %s for song %d: %v", diffStr, id, err)
				}
				continue
			}
			if err := du.db.UpsertChart(du.ctx, chartID, chart); err != nil {
				log.Errorf("failed to upsert chart %s: %v", chartID, err)
			} else {
				log.Infof("updated chart %s", chartID)
			}
		}
	}
	log.Infof("checked jackets and charts for song %d", id)
	return true, nil
}

func (du *DataUpdater) updatePosts() error {
	if err := du.ctx.Err(); err != nil {
		return err
	}
	data, err := files.LoadCache("POST_UPDATE_INFO")
	if err != nil {
		log.Errorf("failed to load post update cache: %v", err)
		return err
	}
	var info PostUpdateInfo
	if len(data) == 0 {
		newestId, err := du.db.GetNewestPostID(du.ctx)
		if err == nil && newestId > 0 {
			info.LastID = newestId
		} else {
			info.LastID = 0
		}
		info.Time = time.Now()
	} else {
		if err := json.Unmarshal(data, &info); err != nil {
			log.Errorf("failed to parse post update cache: %v", err)
			return err
		}
	}

	currentID := info.LastID + 1

	for {
		if err := du.ctx.Err(); err != nil {
			return err
		}
		exists, err := du.UpdatePostByID(currentID)
		if err != nil {
			if !exists {
				log.Warnf("failed to get post %d or it does not exist", currentID)
				currentID++
				if currentID-info.LastID > du.postGapLimit {
					log.Infof("post update stopped, consecutive missing posts exceed %d", du.postGapLimit)
					break
				}
				continue
			}
			log.Errorf("failed to update post %d: %v", currentID, err)
		}
		if exists {
			info.LastID = currentID
			info.Time = time.Now()
			cacheData, _ := json.Marshal(info)
			if err := files.SaveCache("POST_UPDATE_INFO", cacheData); err != nil {
				log.Errorf("failed to save post update cache: %v", err)
			}
		}
		currentID++
	}

	return nil
}

func (du *DataUpdater) UpdatePostByID(id int) (bool, error) {
	if err := du.ctx.Err(); err != nil {
		return false, err
	}
	postInst, err := getPost(du.bestdoriAPI, du.niconiAPI, id)
	if err != nil {
		return false, err
	}
	if postInst.Info.CategoryName == "SELF_POST" && postInst.Info.CategoryId == "chart" {
		log.Infof("updating post %d...", id)
		if err := du.db.UpsertPost(du.ctx, id, postInst.Info); err != nil {
			return true, err
		}
		log.Infof("updated post %d", id)
	}
	return true, nil
}

// StartUpdating schedules periodic updates every 10 minutes and ensures only one run at a time.
func (du *DataUpdater) StartUpdating() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-du.ctx.Done():
				du.waitForRunningUpdate()
				return
			case t := <-ticker.C:
				now := t
				if now.Minute()%10 == 0 {
					du.launchScheduledUpdate()
				}
			}
		}
	}()
}

func (du *DataUpdater) launchScheduledUpdate() {
	du.mu.Lock()
	if du.updateRunning {
		log.Info("previous update still running, skipping this schedule")
		du.mu.Unlock()
		return
	}
	du.updateRunning = true
	done := make(chan struct{})
	du.updateDone = done
	du.mu.Unlock()

	go du.runScheduledUpdate(done)
}

func (du *DataUpdater) runScheduledUpdate(done chan struct{}) {
	defer func() {
		close(done)
		du.mu.Lock()
		du.updateRunning = false
		du.updateDone = nil
		du.mu.Unlock()
	}()

	log.Info("updating job running...")
	if err := du.Update(); err != nil {
		if du.ctx.Err() != nil && err == du.ctx.Err() {
			log.Infof("update canceled: %v", err)
		} else {
			log.Errorf("failed to update: %v", err)
		}
	} else {
		log.Infof("update complete")
	}
}

func (du *DataUpdater) waitForRunningUpdate() {
	du.mu.Lock()
	done := du.updateDone
	running := du.updateRunning
	du.mu.Unlock()

	if running && done != nil {
		<-done
	}
}
