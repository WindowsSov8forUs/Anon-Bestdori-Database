package data

import (
	"sync"
	"time"

	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/dto"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/post"

	"anon-bestdori-database/pkg/log"
)

func (du *DataUpdater) Init() error {
	if err := du.initSongs(); err != nil {
		log.Errorf("failed to initialize songs data: %v", err)
		return err
	}

	if err := du.initPosts(); err != nil {
		log.Errorf("failed to initialize posts data: %v", err)
		return err
	}

	return nil
}

func (du *DataUpdater) initSongs() error {
	return du.updateSongs()
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
			log.Errorf("failed to get post list with offset %d: %v", offset, err)
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
				existing, err := du.db.GetPostByID(du.ctx, pid)
				if err != nil {
					log.Errorf("failed to check existing post %d: %v", pid, err)
					continue
				}
				if existing != nil {
					log.Infof("post %d already exists, skipping initialization", pid)
					continue
				}

				log.Infof("getting info of post %d ...", pid)
				err = retry(func() error {
					postInst, err := getPost(du.bestdoriAPI, du.niconiAPI, pid)
					if err != nil {
						return err
					}
					return du.db.UpsertPost(du.ctx, pid, postInst.Info)
				})
				if err != nil {
					log.Errorf("failed to get info of post %d: %v", pid, err)
				} else {
					log.Infof("initialized post %d", pid)
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
