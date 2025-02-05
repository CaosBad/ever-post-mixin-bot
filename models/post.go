package models

import (
	"context"
	"errors"
	//"github.com/caosbad/ever-post-mixin-bot/routes"
	// "crypto/x509"
	// "encoding/pem"
	"time"

	"github.com/go-pg/pg"

	client "github.com/MixinNetwork/bot-api-go-client"
	"github.com/caosbad/ever-post-mixin-bot/session"
	// "github.com/caosbad/ever-post-mixin-bot/utils"
	// uuid "github.com/satori/go.uuid"
	"github.com/caosbad/telegraph"
)

const posts_DDL = `
CREATE TABLE IF NOT EXISTS posts (
  post_id            	VARCHAR(36) PRIMARY KEY,
  user_id	        	VARCHAR(36) NOT NULL,
  trace_id				VARCHAR(64),
  title          		VARCHAR(256),
  path        			VARCHAR(256), 
  telegraph_url	    	VARCHAR(1024),
  description	    		VARCHAR(1024),
  ipfs_id	    				VARCHAR(64),
  content	    				JSON,
  markdown_content				TEXT,
  created_at        	TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at        	    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
);

CREATE INDEX ON posts (user_id);
`

type Post struct {
	PostId       string    `sql:"post_id,pk"`
	UserId       string    `sql:"user_id,notnull"`
	Title        string    `sql:"title,notnull"`
	Path         string    `sql:"path"`
	TelegraphUrl string    `sql:"telegraph_url"`
	Description  string    `sql:"description"`
	IpfsId       string    `sql:"ipfs_id"`
	TraceId      string    `sql:"trace_id"`
	Content      string    `sql:"content,notnull"`
	CreatedAt    time.Time `sql:"created_at,notnull"`
	UpdatedAt    time.Time `sql:"updated_at,notnull"`
}
type PostListItem struct {
	PostId       string    `sql:"post_id,pk"`
	UserId       string    `sql:"user_id,notnull"`
	Title        string    `sql:"title,notnull"`
	Path         string    `sql:"path"`
	TelegraphUrl string    `sql:"telegraph_url"`
	Description  string    `sql:"description"`
	IpfsId       string    `sql:"ipfs_id"`
	AvatarURL    string    `sql:"avatar_url"`
	Content      string    `sql:"content,notnull"`
	TraceId      string    `sql:"trace_id"`
	CreatedAt    time.Time `sql:"created_at,notnull"`
	UpdatedAt    time.Time `sql:"updated_at,notnull"`
}

// request body
type PostBody struct {
	PostId      string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	HtmlContent string `json:"htmlContent"`
	TraceId     string `json:"traceId"`
	IpfsId      string `json:"ipfsId"`
}

var postCols = []string{"post_id", "title", "description", "content", "path", "user_id", "trace_id", "telegraph_url", "ipfs_id", "created_at"}

func CreateDraft(ctx context.Context, user *User, title, description, content string) (*Post, error) {

	post := &Post{
		PostId:      client.UuidNewV4().String(),
		UserId:      user.UserId,
		Title:       title,
		Description: description,
		Content:     content,
		CreatedAt:   time.Now(),
	}
	if err := session.Database(ctx).Insert(post); err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return post, nil
}

func PublishPost(ctx context.Context, user *User, body PostBody) (*Post, error) {
	// get telegraph account
	account, err := FillTelegraphAccountWithUser(user)
	contentFormated, err := telegraph.ContentFormat(body.HtmlContent)
	if err != nil {
		return nil, err
	}
	page, err := account.CreatePage(&telegraph.Page{
		Title:       body.Title,
		AuthorName:  user.FullName,
		Content:     contentFormated,
		Description: body.Description,
	}, true)

	if err != nil {
		return nil, err
	}

	post := &Post{
		PostId:       body.PostId,
		TelegraphUrl: page.URL,
		Description:  page.Description,
		Path:         page.Path,
		UpdatedAt:    time.Now(),
	}
	if _, err := session.Database(ctx).Model(post).Column("telegraph_url", "path", "updated_at").WherePK().Update(); err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return post, nil
}

func UpdatePost(ctx context.Context, user *User, body PostBody) (*Post, error) {
	post, err := FindPostByPostId(ctx, body.PostId)

	if err != nil || post == nil {
		return nil, errors.New("Post not found.")
	}
	page := &telegraph.Page{
		Path: post.Path,
	}

	// change while props need update
	if body.Title != "" {
		post.Title = body.Title
		page.Title = body.Title
	}
	if body.Description != "" {
		post.Description = body.Description
		page.Description = body.Description
	}
	if body.Content != "" {
		post.Content = body.Content
		post.UpdatedAt = time.Now()
	}
	if body.TraceId != "" {
		post.TraceId = body.TraceId
	}
	if body.HtmlContent != "" {
		contentFormated, err := telegraph.ContentFormat(body.HtmlContent)
		if err != nil {
			return nil, err
		}
		page.Content = contentFormated
	}
	if body.IpfsId != "" {
		post.IpfsId = body.IpfsId
	}

	// edit telegraph page
	if post.Path != "" && user.UserId == post.UserId {
		if account, err := FillTelegraphAccountWithUser(user); err != nil {
			return nil, err
		} else if _, err := account.EditPage(page, true); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("You are not the author of this article.")
	}

	if _, err := session.Database(ctx).Model(post).Column("title", "description", "content", "updated_at", "ipfs_id").WherePK().Update(); err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return post, nil

}
func UpdateDraft(ctx context.Context, user *User, body PostBody) (*Post, error) {
	post, err := FindPostByPostId(ctx, body.PostId)

	if err != nil || post == nil {
		return nil, errors.New("Post not found.")
	}

	// change while props need update
	if body.Title != "" {
		post.Title = body.Title
	}
	if body.Description != "" {
		post.Description = body.Description
	}
	if body.Content != "" {
		post.Content = body.Content
		post.UpdatedAt = time.Now()
	}
	if body.IpfsId != "" {
		post.IpfsId = body.IpfsId
	}
	post.UpdatedAt = time.Now()

	if _, err := session.Database(ctx).Model(post).Column("title", "description", "content", "ipfs_id", "updated_at").WherePK().Update(); err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return post, nil

}

func UpdatePostTraceId(ctx context.Context, post *Post) (*Post, error) {
	if _, err := session.Database(ctx).Model(post).Column("trace_id").WherePK().Update(); err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return post, nil
}

func DeleteDraft(ctx context.Context, user *User, postId string) error {

	if post, err := FindPostByPostId(ctx, postId); err != nil || post.UserId != user.UserId {
		return err
	} else if post.Path != "" || post.IpfsId != "" {
		return errors.New("Post already published.")
	} else if user.UserId != post.UserId {
		return errors.New("You are not the author of this article.")
	} else if _, err = session.Database(ctx).Model(&Post{PostId: postId}).WherePK().Delete(); err != nil {
		return err
	} else {
		return nil
	}
}

func VerifyTrace(ctx context.Context, user *User, traceId string) (*Post, error) {
	var post Post
	if _, err := session.Database(ctx).QueryOne(&post, `SELECT post_id, user_id, trace_id, content, created_at FROM posts WHERE trace_id = ? `, traceId); err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return &post, nil
}

func FindAllPosts(ctx context.Context, offset, limit int) ([]*PostListItem, int, error) {
	var post []*Post
	var posts []*PostListItem
	var orderExp = "created_at DESC"
	if count, err := session.Database(ctx).Model(&post).Order(orderExp).Where("trace_id IS NOT NULL AND path IS NOT NULL").Count(); err != nil {
		return nil, 0, session.TransactionError(ctx, err)
	} else if _, err := session.Database(ctx).Query(&posts, `SELECT P.*, u.avatar_url FROM posts P,users u WHERE P.user_id=u.user_id AND P.trace_id IS NOT NULL AND P.path IS NOT NULL ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset); err == pg.ErrNoRows {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, session.TransactionError(ctx, err)
	} else {
		return posts, count, nil
	}

}

func FindPostsByUser(ctx context.Context, user *User, offset, limit int, postType string) ([]*Post, int, error) {
	var posts []*Post
	var orderExp = "created_at DESC"
	var whereSql string
	switch postType {
	case "draft":
		whereSql = "trace_id IS NULL AND user_id = ?"
	case "ipfs":
		whereSql = "trace_id IS NOT NULL AND path IS NOT NULL AND ipfs_id IS NOT NULL AND user_id = ?"
	case "telegraph":
		whereSql = "trace_id IS NOT NULL AND path IS NOT NULL AND user_id = ?"
	}
	count, err := session.Database(ctx).Model(&posts).Limit(limit).Offset(offset).Order(orderExp).Where(whereSql, user.UserId).SelectAndCount()
	if err == pg.ErrNoRows {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, session.TransactionError(ctx, err)
	}
	return posts, count, nil
}

func FindTelegraphPostsByUser(ctx context.Context, user *User, offset, limit int) (*telegraph.PageList, error) {

	account, err := FillTelegraphAccountWithUser(user)
	if err != nil {
		return nil, err
	}
	list, err := account.GetPageList(offset, limit)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func FindDraftsByUser(ctx context.Context, user *User, offset, limit int) ([]*Post, int, error) {
	var drafts []*Post
	var orderExp = "created_at DESC"
	count, err := session.Database(ctx).Model(&drafts).Limit(limit).Offset(offset).Order(orderExp).Where("trace_id IS NULL AND path IS NULL AND user_id = ?", user.UserId).SelectAndCount()
	if err == pg.ErrNoRows {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, session.TransactionError(ctx, err)
	}
	return drafts, count, nil
}

func FindPostByPath(ctx context.Context, path string) (*Post, error) {
	post := &Post{
		Path: path,
	}
	if _, err := session.Database(ctx).QueryOne(&post, `SELECT post_id, user_id, ipfs_id, path, title, description, created_at FROM posts WHERE path = ?`, path); err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return post, nil
}

func FindPostByPostId(ctx context.Context, postId string) (*Post, error) {
	post := &Post{
		PostId: postId,
	}
	if err := session.Database(ctx).Model(post).Column(postCols...).WherePK().Select(post); err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return post, nil
}

func ListPost(ctx context.Context) ([]*Post, error) {
	// TODO
	var posts []*Post
	err := session.Database(ctx).Model(&posts).Where("client_id IS NOT NULL AND session_id IS NOT NULL AND private_key IS NOT NULL AND expire_at > now()").Select()
	if err != nil {
		return posts, session.TransactionError(ctx, err)
	}
	return posts, nil
}

func FillTelegraphAccountWithUser(user *User) (*telegraph.Account, error) {
	if user.TelegraphToken == "" {
		return nil, errors.New("Telegraph access token not found")
	}
	account := &telegraph.Account{
		AccessToken: user.TelegraphToken,
		ShortName:   user.FullName,
		AuthorName:  user.FullName,
	}
	return account, nil
}
