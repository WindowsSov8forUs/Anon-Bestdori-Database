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
	if err := du.updateSongs(); err != nil {
		log.Errorf("歌曲更新失败: %v", err)
		return err
	}
	if err := du.updatePosts(); err != nil {
		log.Errorf("帖子更新失败: %v", err)
		return err
	}
	return nil
}

func (du *DataUpdater) updateSongs() error {
	all8, err := songs.GetAll8(du.bestdoriAPI)
	if err != nil {
		log.Errorf("获取歌曲 All8 失败: %v", err)
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
		info := (*all8)[strconv.Itoa(id)]
		existing, _ := du.db.GetSongByID(du.ctx, id)
		if !needsSongUpdate(existing, info) {
			continue
		}
		log.Infof("正在更新歌曲 %d 信息...", id)
		song, err := getSong(du.bestdoriAPI, id)
		if err != nil {
			log.Errorf("获取歌曲 %d 失败: %v", id, err)
			continue
		}
		if err := du.db.UpsertSong(du.ctx, id, song.Info); err != nil {
			log.Errorf("更新歌曲 %d 失败: %v", id, err)
			continue
		}
		log.Infof("已更新歌曲 %d", id)

		// 更新 music jackets，只当 jacketImage 文件缺失时（更改/新增）
		jackets := song.GetJacket()
		for _, jacket := range jackets {
			jacketName := fmt.Sprintf("musicjacket/%s.png", jacket.JacketImage)
			if _, err := files.GetAssets(jacketName); err != nil {
				log.Infof("下载缺失的 jacket %s for song %d", jacket.JacketImage, id)
				if err := retry(func() error {
					return downloadMusicJacket(jacket)
				}); err != nil {
					log.Errorf("更新歌曲 %d jacket %s 失败: %v", id, jacket.JacketImage, err)
				} else {
					log.Infof("已更新 jacket %s for song %d", jacket.JacketImage, id)
				}
			}
		}

		// 更新 charts
		diffs := []string{"easy", "normal", "hard", "expert", "special"}
		for _, diffStr := range diffs {
			chartID := fmt.Sprintf("%d-%s", id, diffStr)
			if existingChart, _ := du.db.GetChartByID(du.ctx, chartID); existingChart == nil {
				log.Infof("更新缺失 chart %s for song %d", chartID, id)
				diff := dto.ChartDifficultyName(diffStr)
				chart, err := getChart(song, diff)
				if err != nil {
					if _, ok := err.(*bestdori.NotExistError); !ok {
						log.Errorf("获取歌曲 %d %s 谱面失败: %v", id, diffStr, err)
					}
					continue
				}
				if err := du.db.UpsertChart(du.ctx, chartID, chart); err != nil {
					log.Errorf("更新谱面 %s 失败: %v", chartID, err)
				} else {
					log.Infof("已更新谱面 %s", chartID)
				}
			}
		}
		log.Infof("歌曲 %d 的 jacket 和 charts 已检查更新", id)
	}
	return nil
}

func (du *DataUpdater) updatePosts() error {
	data, err := files.LoadCache("POST_UPDATE_INFO")
	if err != nil {
		log.Errorf("加载帖子更新缓存失败: %v", err)
		return err
	}
	var info PostUpdateInfo
	if len(data) == 0 {
		// 初始化
		newestId, err := du.db.GetNewestPostID(du.ctx)
		if err == nil && newestId > 0 {
			info.LastID = newestId
		} else {
			info.LastID = 0
		}
		info.Time = time.Now()
	} else {
		if err := json.Unmarshal(data, &info); err != nil {
			log.Errorf("解析帖子更新缓存失败: %v", err)
			return err
		}
	}

	currentID := info.LastID + 1

	for {
		postInst, err := getPost(du.bestdoriAPI, du.niconiAPI, currentID)
		if err != nil {
			log.Warnf("帖子 %d 获取失败或不存在", currentID)
			currentID++
			if currentID-info.LastID > du.postGapLimit {
				log.Infof("帖子更新停止，连续空缺超过 %d", du.postGapLimit)
				break
			}
			continue
		}

		// 更新缓存
		info.LastID = currentID
		info.Time = time.Now()
		cacheData, _ := json.Marshal(info)
		if err := files.SaveCache("POST_UPDATE_INFO", cacheData); err != nil {
			log.Errorf("保存帖子更新缓存失败: %v", err)
		}

		info.LastID = currentID

		if postInst.Info.CategoryName == "SELF_POST" && postInst.Info.CategoryId == "chart" {
			log.Infof("正在更新帖子 %d...", currentID)
			if err := du.db.UpsertPost(du.ctx, currentID, postInst.Info); err != nil {
				log.Errorf("更新帖子 %d 失败: %v", currentID, err)
			} else {
				log.Infof("已更新帖子 %d", currentID)
			}
		}

		currentID++
	}

	return nil
}

// StartUpdating 启动定时更新调度器，每分钟检查一次，在分钟数为整10时运行 Update，且保证只有一个同时运行
func (du *DataUpdater) StartUpdating() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-du.ctx.Done():
				return
			case t := <-ticker.C:
				now := t
				if now.Minute()%10 == 0 {
					du.mu.Lock()
					defer du.mu.Unlock()
					log.Info("updating job running...")
					if err := du.Update(); err != nil {
						log.Errorf("failed to update: %v", err)
					} else {
						log.Infof("update complete")
					}
				}
			}
		}
	}()
}
