package boot

import (
	"os"

	"go-gin-gorm-example/infrastructure/config"
	"go-gin-gorm-example/infrastructure/database"
	"go-gin-gorm-example/infrastructure/limiter"
	"go-gin-gorm-example/infrastructure/listener"
	logger "go-gin-gorm-example/infrastructure/log"
	"go-gin-gorm-example/infrastructure/redis"
	"go-gin-gorm-example/module/article"
	"go-gin-gorm-example/module/health"
	"go-gin-gorm-example/utils"

	redisThirdPartyLib "github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

type HandlerSetup struct {
	Limiter     *limiter.RateLimiter
	HealthHttp  health.InterfaceHttp
	ArticleHttp article.InterfaceHttp
}

func MakeHandler() HandlerSetup {
	//initiate config
	config.Initialize()

	//initiate logger
	logger.Init(config.Conf.LogFormat, config.Conf.LogLevel)

	var err error

	//initiate a redis client
	var redisClient *redisThirdPartyLib.Client
	var redisLibInterface redis.LibInterface
	if config.Conf.Redis.EnableRedis {
		redisClient, err = redis.NewRedisClient(&config.Conf)
		if err != nil {
			log.Fatalf("failed initiate redis: %v", err)
			os.Exit(1)
		}
		//initiate a redis library interface
		redisLibInterface, err = redis.NewRedisLibInterface(redisClient)
		if err != nil {
			log.Fatalf("failed initiate redis library: %v", err)
			os.Exit(1)
		}
	}

	//setup infrastructure postgres
	var db database.HandlerDatabase
	if config.Conf.Postgres.EnablePostgres {
		db, err = database.NewDatabaseClient(&config.Conf)
		if err != nil {
			log.Fatalf("failed initiate database postgres: %v", err)
			os.Exit(1)
		}
	}

	//add limiter
	interval := utils.StringUnitToDuration(config.Conf.Interval)
	middlewareWithLimiter := limiter.NewRateLimiter(int(config.Conf.Rate), interval)

	//health module
	//article module
	var articleRepository article.RepositoryInterface
	var healthRepository health.RepositoryInterface
	if config.Conf.Postgres.EnablePostgres {
		articleRepository = article.NewRepository(db.DbConn)
		healthRepository = health.NewRepository(db.DbConn)
	} else {
		articleRepository = article.NewInMemoryRepositoryRepositoryAdapter()
	}

	healthService := health.NewService(healthRepository, redisClient)
	healthModule := health.NewHttp(healthService)

	articleService := article.NewService(articleRepository, redisLibInterface)
	articleModule := article.NewHttp(articleService)

	// add listener instance
	listen := listener.NewListener(articleModule)
	//add for trigger start up
	listen.TriggerStartUp()
	//add listen for shutdown event
	listen.ListenForShutdownEvent()

	return HandlerSetup{
		Limiter:     middlewareWithLimiter,
		HealthHttp:  healthModule,
		ArticleHttp: articleModule,
	}
}
