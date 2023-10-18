package article

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"go-gin-gorm-example/infrastructure/httplib"
	logger "go-gin-gorm-example/infrastructure/log"
	"go-gin-gorm-example/infrastructure/validator"
	"go-gin-gorm-example/module/primitive"
	"go-gin-gorm-example/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Http struct {
	serviceArticle InterfaceService
}

func NewHttp(serviceHealth InterfaceService) InterfaceHttp {
	return &Http{
		serviceArticle: serviceHealth,
	}
}

type InterfaceHttp interface {
	GroupArticle(group *gin.RouterGroup)
	SaveToFile()
	LoadFromFile()
}

func (h *Http) GroupArticle(g *gin.RouterGroup) {
	g.GET("", h.GetListArticle)
	g.GET("/:id", h.DetailArticle)
	g.POST("", h.CreateArticle)
}

func (h *Http) GetListArticle(c *gin.Context) {
	logCtx := fmt.Sprintf("handler.GetListArticle")
	ctx := context.Background()

	if h.serviceArticle == nil {
		err := errors.New("dependency service article to handler article on method GetListArticle is nil")
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "h.serviceHealth")
		httplib.SetErrorResponse(c, http.StatusInternalServerError, primitive.SomethingWentWrong)
		return
	}

	paginationQuery, err := httplib.GetPaginationFromCtx(c)
	if err != nil {
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "httplib.GetPaginationFromCtx")
		httplib.SetErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	query := c.Request.URL.Query().Get("query")
	if query != "" {
		if !utils.IsValidSanitizeSQL(query) {
			err = errors.New(primitive.QueryIsSuspicious)
			logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "utils.IsValidSanitizeSQL")
			httplib.SetErrorResponse(c, http.StatusBadRequest, primitive.QueryIsSuspicious)
			return
		}
	}

	author := c.Request.URL.Query().Get("author")
	if author != "" {
		if !utils.IsValidSanitizeSQL(author) {
			err = errors.New(primitive.QueryIsSuspicious)
			logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "utils.IsValidSanitizeSQL")
			httplib.SetErrorResponse(c, http.StatusBadRequest, primitive.QueryIsSuspicious)
			return
		}
	}

	param := primitive.ParameterArticleHandler{
		Query:  query,
		Author: author,
	}

	data, count, err := h.serviceArticle.GetListArticle(ctx, param, paginationQuery)
	if err != nil {
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "h.serviceArticle.GetListArticle")
		httplib.SetErrorResponse(c, http.StatusInternalServerError, primitive.SomethingWentWrong)
		return
	}

	httplib.SetPaginationResponse(c,
		http.StatusOK,
		primitive.SuccessGetArticle,
		data,
		uint64(count),
		paginationQuery)
	return
}

func (h *Http) CreateArticle(c *gin.Context) {
	logCtx := fmt.Sprintf("handler.CreateArticle")
	ctx := context.Background()

	if h.serviceArticle == nil {
		err := errors.New("dependency service article to handler article on method CreateArticle is nil")
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "h.serviceHealth")
		httplib.SetErrorResponse(c, http.StatusInternalServerError, primitive.SomethingWentWrong)
		return
	}

	var requestBody primitive.ArticleReq
	// Decode the request body into the Article struct.
	if err := c.ShouldBind(&requestBody); err != nil {
		httplib.SetErrorResponse(c, http.StatusBadRequest, primitive.SomethingWrongWithTheBodyRequest)
		return
	}

	errValidateStruct := validator.ValidateStructResponseSliceString(requestBody)
	if errValidateStruct != nil {
		logger.Error(ctx, logCtx, "validator.ValidateStructResponseSliceString got err : %v", errValidateStruct)
		httplib.SetCustomResponse(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil, errValidateStruct)
		return
	}

	data, err := h.serviceArticle.RecordArticle(ctx, requestBody)
	if err != nil {
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "h.serviceArticle.GetListArticle")
		httplib.SetErrorResponse(c, http.StatusInternalServerError, primitive.SomethingWentWrong)
		return
	}

	httplib.SetSuccessResponse(c, http.StatusOK, primitive.SuccessCreateArticle, data)
	return

}

func (h *Http) DetailArticle(c *gin.Context) {
	logCtx := fmt.Sprintf("handler.DetailArticle")
	ctx := context.Background()

	if h.serviceArticle == nil {
		err := errors.New("dependency service article to handler article on method DetailArticle is nil")
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "h.serviceHealth")
		httplib.SetErrorResponse(c, http.StatusInternalServerError, primitive.SomethingWentWrong)
		return
	}

	idParam := c.Param("id")
	if idParam == "" {
		err := errors.New(primitive.ParamIdIsZeroOrNullString)
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "c.Param")
		httplib.SetErrorResponse(c, http.StatusBadRequest, primitive.ParamIdIsZeroOrNullString)
		return
	}

	idInt64, err := strconv.Atoi(idParam)
	if err != nil || idInt64 == 0 {
		err := errors.New(primitive.ParamIdIsZeroOrNullString)
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "strconv.Atoi")
		httplib.SetErrorResponse(c, http.StatusBadRequest, primitive.ParamIdIsZeroOrNullString)
		return
	}

	data, err := h.serviceArticle.GetDetailArticle(ctx, int64(idInt64))
	if err != nil {
		errNotFound := []error{gorm.ErrRecordNotFound, primitive.ErrorArticleNotFound}
		if utils.ContainsError(err, errNotFound) {
			logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "h.serviceArticle.GetDetailArticle")
			httplib.SetErrorResponse(c, http.StatusNotFound, primitive.RecordArticleNotFound)
			return
		}
		logger.Error(ctx, utils.ErrorLogFormat, err.Error(), logCtx, "h.serviceArticle.GetDetailArticle")
		httplib.SetErrorResponse(c, http.StatusInternalServerError, primitive.SomethingWentWrong)
		return
	}

	httplib.SetSuccessResponse(c, http.StatusOK, primitive.SuccessCreateArticle, data)
	return

}

func (h *Http) SaveToFile() {
	ctx := context.Background()
	h.serviceArticle.RecordArticleToFile(ctx)
}

func (h *Http) LoadFromFile() {
	ctx := context.Background()
	h.serviceArticle.LoadArticleToFile(ctx)
}
