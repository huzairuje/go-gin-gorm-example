package router

import (
	"github.com/gin-gonic/gin"
	"go-gin-gorm-example/infrastructure/config"
	"net/http"

	"go-gin-gorm-example/boot"
	"go-gin-gorm-example/infrastructure/httplib"
	"go-gin-gorm-example/infrastructure/middleware"
)

type HandlerRouter struct {
	Setup boot.HandlerSetup
}

func NewHandlerRouter(setup boot.HandlerSetup) InterfaceRouter {
	return &HandlerRouter{
		Setup: setup,
	}
}

type InterfaceRouter interface {
	RouterWithMiddleware() *gin.Engine
}

func notFoundHandler(c *gin.Context) {
	// render 404 custom response
	httplib.SetErrorResponse(c, http.StatusNotFound, "Not Matching of Any Routes")
	return
}

func methodNotAllowedHandler(c *gin.Context) {
	// render 404 custom response
	httplib.SetErrorResponse(c, http.StatusMethodNotAllowed, "Method Not Allowed")
	return
}

func (hr *HandlerRouter) RouterWithMiddleware() *gin.Engine {
	//add new instance for bun router and add not found handler
	//and method with not allowed handler
	c := gin.New()

	//use recovery
	c.Use(gin.Recovery())

	//if logMode is true set logger to stdout on gin
	if config.Conf.LogMode {
		c.Use(gin.Logger())
	}

	//set middleware to use not found handler
	c.NoRoute(notFoundHandler)

	//set middleware to use method not allowed
	c.NoMethod(methodNotAllowedHandler)

	//grouping on root endpoint
	api := c.Group("/api")

	api.Use(middleware.RateLimiterMiddleware(hr.Setup.Limiter))

	//grouping on "api/v1"
	v1 := api.Group("/v1")

	//module health
	prefixHealth := v1.Group("/health")
	hr.Setup.HealthHttp.GroupHealth(prefixHealth)

	//module article
	prefixArticle := v1.Group("/articles")
	hr.Setup.ArticleHttp.GroupArticle(prefixArticle)

	return c

}
