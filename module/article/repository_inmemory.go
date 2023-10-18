package article

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"go-gin-gorm-example/module/primitive"
)

// InMemoryRepository is an in-memory implementation of the RepositoryInterface.
type InMemoryRepository struct {
	articles   []primitive.Article
	idSequence int64
	mu         sync.RWMutex
}

// NewInMemoryRepository creates a new instance of InMemoryRepository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		articles:   make([]primitive.Article, 0),
		idSequence: 1,
	}
}

// NewInMemoryRepositoryRepositoryAdapter creates a new instance of RepositoryInterface using InMemoryRepository.
func NewInMemoryRepositoryRepositoryAdapter() RepositoryInterface {
	return NewInMemoryRepository()
}

func (r *InMemoryRepository) CreateArticle(ctx context.Context, payload primitive.Article) (primitive.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	payload.ID = nextID(r.articles)
	payload.CreatedAt = time.Now()
	payload.UpdatedAt = time.Now()
	r.idSequence++

	r.articles = append(r.articles, payload)
	return payload, nil
}

func (r *InMemoryRepository) CountArticle(ctx context.Context, param primitive.ParameterFindArticle) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, article := range r.articles {
		if article.DeletedAt.IsZero() && (param.Author == "" || strings.Contains(article.Author, param.Author)) &&
			(param.Query == "" || strings.Contains(article.Title, param.Query) || strings.Contains(article.Body, param.Query)) {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryRepository) FindListArticle(ctx context.Context, param primitive.ParameterFindArticle) ([]primitive.Article, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var listData []primitive.Article
	for _, article := range r.articles {
		if article.DeletedAt.IsZero() && (param.Author == "" || strings.Contains(article.Author, param.Author)) &&
			(param.Query == "" || strings.Contains(article.Title, param.Query) || strings.Contains(article.Body, param.Query)) {
			listData = append(listData, article)
		}
	}

	// Apply sorting
	sortField := r.SetParamQueryToOrderByQuery(param.SortBy)
	if param.SortOrder == "desc" {
		sortField = "-" + sortField
	}

	listData = sortArticles(listData, sortField)

	// Apply pagination
	startIdx := param.Offset
	endIdx := param.Offset + param.PageSize
	if endIdx > len(listData) {
		endIdx = len(listData)
	}
	return listData[startIdx:endIdx], nil
}

func (r *InMemoryRepository) SetParamQueryToOrderByQuery(orderBy string) string {
	switch orderBy {
	case "id":
		return "ID"
	case "author":
		return "Author"
	case "title":
		return "Title"
	case "body":
		return "Body"
	case "created":
		return "CreatedAt"
	default:
		return "CreatedAt"
	}
}

func (r *InMemoryRepository) FindArticleByID(ctx context.Context, articleID int64) (primitive.Article, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, article := range r.articles {
		if article.ID == articleID && article.DeletedAt.IsZero() {
			return article, nil
		}
	}
	return primitive.Article{}, primitive.ErrorArticleNotFound
}

// SaveToFile saves the articles data to a JSON file.
func (r *InMemoryRepository) SaveToFile(filePath string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.articles, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// LoadFromFile loads articles data from a JSON file.
func (r *InMemoryRepository) LoadFromFile(filePath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var articles []primitive.Article
	err = json.Unmarshal(data, &articles)
	if err != nil {
		return err
	}

	r.articles = articles

	return nil
}

// Helper function to sort articles based on a given field.
func sortArticles(articles []primitive.Article, field string) []primitive.Article {
	switch field {
	case "ID":
		return sortByID(articles)
	case "Author":
		return sortByAuthor(articles)
	case "Title":
		return sortByTitle(articles)
	case "Body":
		return sortByBody(articles)
	case "CreatedAt":
		return sortByCreatedAt(articles)
	default:
		return sortByCreatedAt(articles)
	}
}

func sortByID(articles []primitive.Article) []primitive.Article {
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].ID > articles[j].ID
	})
	return articles
}

func sortByAuthor(articles []primitive.Article) []primitive.Article {
	sort.Slice(articles, func(i, j int) bool {
		return strings.Compare(articles[i].Author, articles[j].Author) < 0
	})
	return articles
}

func sortByTitle(articles []primitive.Article) []primitive.Article {
	sort.Slice(articles, func(i, j int) bool {
		return strings.Compare(articles[i].Title, articles[j].Title) < 0
	})
	return articles
}

func sortByBody(articles []primitive.Article) []primitive.Article {
	sort.Slice(articles, func(i, j int) bool {
		return strings.Compare(articles[i].Body, articles[j].Body) < 0
	})
	return articles
}

func sortByCreatedAt(articles []primitive.Article) []primitive.Article {
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].CreatedAt.Before(articles[j].CreatedAt)
	})
	return articles
}

// Helper function to determine the next ID based on the highest ID in the loaded articles
func nextID(articles []primitive.Article) int64 {
	var maxID int64
	for _, article := range articles {
		if article.ID > maxID {
			maxID = article.ID
		}
	}
	return maxID + 1
}

func findMaxID(articles []primitive.Article) int64 {
	var maxID int64
	for _, article := range articles {
		if article.ID > maxID {
			maxID = article.ID
		}
	}
	return maxID
}
