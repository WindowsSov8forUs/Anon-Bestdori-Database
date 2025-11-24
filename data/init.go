package data

import (
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/dto"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/post"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/songs"

	"anon-bestdori-database/pkg/log"
)

func (du *DataUpdater) Init() error {
	if err := du.initSongs(); err != nil {
		log.Errorf("歌曲初始化失败: %v", err)
		return err
	}

	if err := du.initPosts(); err != nil {
		log.Errorf("帖子初始化失败: %v", err)
		return err
	}

	return nil
}

func (du *DataUpdater) initSongs() error {
	all0, err := songs.GetAll0(du.bestdoriAPI)
	if err != nil {
		log.Errorf("获取歌曲 ID 列表失败: %v", err)
		return err
	}

	// 获取歌曲 ID 列表并排序
	idList := make([]int, 0, len(*all0))
	for idStr := range *all0 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		idList = append(idList, id)
	}
	slices.Sort(idList)

	for _, id := range idList {
		log.Infof("正在获取歌曲 %d 信息...", id)

		song, err := getSong(du.bestdoriAPI, id)
		if err != nil {
			log.Errorf("获取歌曲 %d 失败: %v", id, err)
			return err
		}
		if err := du.db.UpsertSong(du.ctx, id, song.Info); err != nil {
			log.Errorf("插入歌曲 %d 失败: %v", id, err)
			return err
		}

		// 下载音乐封面
		jackets := song.GetJacket()
		for _, jacket := range jackets {
			err := retry(func() error {
				return downloadMusicJacket(jacket)
			})
			if err != nil {
				log.Errorf("下载歌曲 %d 封面失败: %v", id, err)
			}
		}
		// 下载音频
		err = retry(func() error {
			return downloadBGM(song)
		})
		if err != nil {
			log.Errorf("下载歌曲 %d 音频失败: %v", id, err)
		}

		log.Infof("已初始化歌曲 %d", id)

		// 初始化谱面
		diffs := []string{"easy", "normal", "hard", "expert", "special"}
		for _, diffStr := range diffs {
			diff := dto.ChartDifficultyName(diffStr)
			chart, err := getChart(song, diff)
			if err != nil {
				if _, ok := err.(*bestdori.NotExistError); !ok {
					log.Errorf("获取歌曲 %d %s 谱面失败: %v", id, diffStr, err)
					return err
				}
				continue
			}
			chartID := fmt.Sprintf("%d-%s", id, diffStr)
			if err := du.db.UpsertChart(du.ctx, chartID, chart); err != nil {
				log.Errorf("插入谱面 %s 失败: %v", chartID, err)
				return err
			}
		}
		log.Infof("已初始化歌曲 %d 所有谱面", id)
	}
	return nil
}

func getPostList(du *DataUpdater, offset, limit int) (*dto.PostList, error) {
	var list *dto.PostList
	err := retry(func() error {
		var err error
		list, err = post.GetList(
			du.bestdoriAPI,
			"",
			false,
			"SELF_POST",
			"chart",
			nil,
			"",
			post.OrderTimeAsc,
			limit,
			offset,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (du *DataUpdater) initPosts() error {
	var wg sync.WaitGroup
	offset := 0
	limit := 50

	for {
		list, err := getPostList(du, offset, limit)
		if err != nil {
			log.Errorf("获取帖子列表 offset=%d 失败: %v", offset, err)
			return err
		}

		if len(list.Posts) == 0 {
			break
		}

		postIDs := make([]int, len(list.Posts))
		for i, p := range list.Posts {
			postIDs[i] = p.Id
		}

		wg.Add(1)
		go func(ids []int) {
			defer wg.Done()
			for _, pid := range ids {
				log.Infof("正在获取帖子 %d 信息...", pid)
				err := retry(func() error {
					postInst, err := getPost(du.bestdoriAPI, du.niconiAPI, pid)
					if err != nil {
						return err
					}
					return du.db.UpsertPost(du.ctx, pid, postInst.Info)
				})
				if err != nil {
					log.Errorf("处理帖子 %d 失败: %v", pid, err)
				} else {
					log.Infof("已初始化帖子 %d", pid)
				}
			}
		}(postIDs)

		time.Sleep(3 * time.Second)
		offset += limit

		if len(list.Posts) < limit {
			break
		}
	}

	wg.Wait()
	return nil
}
