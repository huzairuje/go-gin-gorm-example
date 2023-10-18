package article

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-gin-gorm-example/infrastructure/config"
	"go-gin-gorm-example/infrastructure/httplib"
	logger "go-gin-gorm-example/infrastructure/log"
	"go-gin-gorm-example/infrastructure/redis"
	"go-gin-gorm-example/module/primitive"
	"go-gin-gorm-example/utils"
)

const (
	redisFinaleKeyArticle     = "article:%d"
	redisListFinaleKeyArticle = "article_list"
)

type InterfaceService interface {
	GetListArticle(ctx context.Context, param primitive.ParameterArticleHandler, pagination *httplib.Query) (resp []primitive.ArticleResp, count int64, err error)
	RecordArticle(ctx context.Context, payload primitive.ArticleReq) (primitive.ArticleResp, error)
	GetDetailArticle(ctx context.Context, articleID int64) (primitive.ArticleResp, error)
	RecordArticleToFile(ctx context.Context)
	LoadArticleToFile(ctx context.Context)
}

type Service struct {
	repository RepositoryInterface
	redis      redis.LibInterface
}

func NewService(repository RepositoryInterface, redisLib redis.LibInterface) InterfaceService {
	if repository == nil {
		panic("repository is not implemented!")
	}
	return &Service{
		repository: repository,
		redis:      redisLib,
	}
}

func (s Service) RecordArticle(ctx context.Context, payload primitive.ArticleReq) (primitive.ArticleResp, error) {
	logCtx := fmt.Sprintf("service.RecordArticle")

	payloadDb := primitive.Article{
		Author: payload.Author,
		Title:  payload.Title,
		Body:   payload.Body,
	}

	data, err := s.repository.CreateArticle(ctx, payloadDb)
	if err != nil {
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "u.repository.CountArticle")
		return primitive.ArticleResp{}, err
	}

	//set data to redis on goroutine
	if config.Conf.Redis.EnableRedis && s.redis != nil {
		go func() {
			dataBytes, errMarshall := json.Marshal(data)
			if errMarshall != nil {
				logger.Error(ctx, utils.ErrorLogFormat, errMarshall.Error(), logCtx, "json.Marshal")
			}
			redisFinaleKey := fmt.Sprintf(redisFinaleKeyArticle, data.ID)
			errSetToRedis := s.redis.Set(redisFinaleKey, dataBytes, time.Minute)
			if errSetToRedis != nil {
				logger.Error(ctx, utils.ErrorLogFormat, errSetToRedis.Error(), logCtx, "s.redis.Set")
			}
			fmt.Printf("success SET on redis by key: %s\n", redisFinaleKey)
		}()
	}

	payloadResp := primitive.ArticleResp{
		ID:        data.ID,
		Author:    data.Author,
		Title:     data.Title,
		Body:      data.Body,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}

	return payloadResp, nil

}

func (s Service) GetListArticle(ctx context.Context, param primitive.ParameterArticleHandler, pagination *httplib.Query) (resp []primitive.ArticleResp, count int64, err error) {
	logCtx := fmt.Sprintf("service.GetListArticle")

	emptySliceDataArticle := make([]primitive.ArticleResp, 0)

	paramQuery := primitive.ParameterFindArticle{
		Query:     param.Query,
		Author:    param.Author,
		PageSize:  pagination.GetSize(),
		Offset:    pagination.GetOffset(),
		SortBy:    s.repository.SetParamQueryToOrderByQuery(pagination.GetOrderBy()),
		SortOrder: pagination.GetSortOrder(),
	}

	// Generate a unique cache key based on the pagination parameters
	cacheKey := fmt.Sprintf("%s:%s:%s:%d:%d:%s:%s",
		redisListFinaleKeyArticle,
		paramQuery.Query,
		paramQuery.Author,
		paramQuery.PageSize,
		paramQuery.Offset,
		paramQuery.SortBy,
		paramQuery.SortOrder)

	// Check if the data exists in the Redis cache
	if config.Conf.Redis.EnableRedis && s.redis != nil {
		cacheData := s.redis.Get(cacheKey)
		if cacheData != "" {
			// If data exists in cache, decode it and return
			if err := json.Unmarshal([]byte(cacheData), &resp); err != nil {
				logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "json.Unmarshal")
			}
			count = int64(len(resp))
			return resp, count, nil
		}
	}

	// Data not found in cache, query the database
	count, err = s.repository.CountArticle(ctx, paramQuery)
	if err != nil {
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "u.repository.CountArticle")
		return
	}

	listData, err := s.repository.FindListArticle(ctx, paramQuery)
	if err != nil {
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "u.repository.FindListArticle")
		return
	}

	if count == 0 && len(listData) == 0 {
		return emptySliceDataArticle, 0, nil
	}

	var list []primitive.ArticleResp
	if len(listData) > 0 {
		for _, val := range listData {
			list = append(list, primitive.ArticleResp{
				ID:        val.ID,
				Author:    val.Author,
				Title:     val.Title,
				Body:      val.Body,
				CreatedAt: val.CreatedAt,
				UpdatedAt: val.UpdatedAt,
			})
		}
		resp = list
	}

	// Store data in Redis cache for next time
	if config.Conf.Redis.EnableRedis && s.redis != nil {
		if len(resp) > 0 {
			go func() {
				cacheDataBytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					logger.Error(ctx, utils.ErrorLogFormat, errMarshal.Error(), logCtx, "json.Marshal")
				}
				// Cache data for a reasonable amount of time (e.g., 1 hour)
				errSetDataRedis := s.redis.Set(cacheKey, cacheDataBytes, time.Minute)
				if errSetDataRedis != nil {
					logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "s.redis.Set")
				}
				fmt.Printf("success SET on redis by key: %s\n", cacheKey)
			}()
		}
	}

	return resp, count, nil
}

func (s Service) GetDetailArticle(ctx context.Context, articleID int64) (primitive.ArticleResp, error) {
	logCtx := fmt.Sprintf("service.GetDetailArticle")

	var resp primitive.ArticleResp
	cacheKey := fmt.Sprintf(redisFinaleKeyArticle, articleID)

	// Check if the data exists in the Redis cache
	if config.Conf.Redis.EnableRedis && s.redis != nil {
		cacheData := s.redis.Get(cacheKey)
		if cacheData != "" {
			// If data exists in cache, decode it and return
			err := json.Unmarshal([]byte(cacheData), &resp)
			if err != nil {
				logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "json.Unmarshal")
			}
			return resp, nil
		}
	}

	data, err := s.repository.FindArticleByID(ctx, articleID)
	if err != nil {
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "u.repository.FindListArticle")
		return primitive.ArticleResp{}, err
	}

	if config.Conf.Redis.EnableRedis && s.redis != nil {
		if data.ID > 0 {
			go func() {
				cacheDataBytes, errMarshal := json.Marshal(data)
				if errMarshal != nil {
					logger.Error(ctx, utils.ErrorLogFormat, errMarshal.Error(), logCtx, "json.Marshal")
				}
				// Cache data for a reasonable amount of time (e.g., 1 hour)
				errSetDataRedis := s.redis.Set(cacheKey, cacheDataBytes, time.Minute)
				if errSetDataRedis != nil {
					logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "s.redis.Set")
				}
				fmt.Printf("success SET on redis by key: %s\n", cacheKey)
			}()
		}
	}

	resp = primitive.ArticleResp{
		ID:        data.ID,
		Author:    data.Author,
		Title:     data.Title,
		Body:      data.Body,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}

	return resp, nil

}

func (s Service) RecordArticleToFile(ctx context.Context) {
	logCtx := fmt.Sprintf("service.RecordArticleToFile")
	if !config.Conf.Postgres.EnablePostgres {
		err := s.repository.SaveToFile("article.json")
		if err != nil {
			logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "s.repository.SaveToFile")
		}
	}
}

func (s Service) LoadArticleToFile(ctx context.Context) {
	logCtx := fmt.Sprintf("service.LoadArticleToFile")
	if !config.Conf.Postgres.EnablePostgres {
		err := s.repository.LoadFromFile("article.json")
		if err != nil {
			logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "s.repository.LoadFromFile")
		}
	}
}
