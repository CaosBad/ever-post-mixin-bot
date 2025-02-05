package routes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MixinNetwork/bot-api-go-client"
	"net/url"

	//"github.com/crossle/hacker-news-mixin-bot/config"
	"github.com/caosbad/ever-post-mixin-bot/config"
	"github.com/caosbad/ever-post-mixin-bot/middlewares"
	"github.com/caosbad/ever-post-mixin-bot/models"
	"github.com/caosbad/ever-post-mixin-bot/session"
	"github.com/caosbad/ever-post-mixin-bot/views"
	"github.com/dimfeld/httptreemux"
	"github.com/satori/go.uuid"
	"net/http"
	"strconv"
)

type postsImpl struct{}

func registerPosts(router *httptreemux.TreeMux) {
	impl := &postsImpl{}
	router.GET("/posts/:id", impl.getPost)
	router.PUT("/posts/:id", impl.updatePost)
	router.POST("/posts/:id", impl.publishPost)
	router.GET("/verify/:id", impl.verifyTrace)
	router.GET("/myPosts/:type", impl.getUserPosts)
	router.GET("/posts", impl.getAllPosts)

	router.PUT("/drafts/:id", impl.updateDraft)
	router.POST("/drafts", impl.createDraft)
	router.DELETE("/drafts/:id", impl.deleteDraft)
	router.GET("/drafts/:id", impl.getUserDraft)
	router.GET("/drafts", impl.getUserDrafts)
	router.POST("/notify/:id", impl.notifyAuthor)

	// router.GET("/posts/:id", impl.getPost)
}

func (impl *postsImpl) createDraft(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body struct {
		Title           string `json:"title"`
		Description     string `json:"description"`
		Content         string `json:"content"`
		MarkdownContent string `json:"markdown"`
		TraceId         string `json:"traceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}
	current := middlewares.CurrentUser(r)
	// postId := params["id"]
	if post, err := models.CreateDraft(r.Context(), current, body.Title, body.Description, body.Content); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderPost(w, r, post)
	}
}

func (impl *postsImpl) publishPost(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var isFirstPublished = false
	postId := params["id"]
	if _, err := uuid.FromString(postId); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	} else if post, err := models.FindPostByPostId(r.Context(), postId); err != nil || post == nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
	} else if post != nil {
		if post.Path == "" {
			isFirstPublished = true
		}
		var body models.PostBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TraceId != post.TraceId {
			views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
			return
		}
		current := middlewares.CurrentUser(r)
		body.PostId = post.PostId
		if post, err := models.PublishPost(r.Context(), current, body); err != nil {
			views.RenderErrorResponse(w, r, err)
		} else {
			if isFirstPublished {
				sendNotifyToSubscribers(r.Context(), post)
			}
			views.RenderPost(w, r, post)
		}
	}

}
func (impl *postsImpl) notifyAuthor(w http.ResponseWriter, r *http.Request, params map[string]string) {
	postId := params["id"]
	var body struct {
		CommentId string `json:"commentId"`
		Text      string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && body.CommentId != "" {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	} else if _, err := uuid.FromString(postId); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	} else if post, err := models.FindPostByPostId(r.Context(), postId); err != nil || post == nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
	} else {
		content := "you have a new comment on post :" + post.Title + " ———— " + " \n 「" + body.Text + "」\n see it here " + "http://everpost.one/post/" + post.PostId
		conversationId := bot.UniqueConversationId(config.ClientId, post.UserId)
		data := base64.StdEncoding.EncodeToString([]byte(content))
		err := bot.PostMessage(r.Context(), conversationId, post.UserId, bot.UuidNewV4().String(), "PLAIN_TEXT", data, config.ClientId, config.SessionId, config.PrivateKey)
		if err != nil {
			fmt.Print(err)
		}
	}

}

func sendNotifyToSubscribers(ctx context.Context, post *models.Post) {
	if subscribers, err := models.ListSubscribers(ctx, post.UserId); err != nil {
		return
	} else if user, err := models.FindUserById(ctx, post.UserId); err != nil {
		return
	} else {
		content := user.FullName + " : " + post.Title + " —— " + "http://everpost.one/post/" + post.PostId
		if post.IpfsId != "" {
			content += " \n " + "https://ipfs.io/ipfs/" + post.IpfsId
		}
		for _, sub := range subscribers {
			conversationId := bot.UniqueConversationId(config.ClientId, sub.SubscriberId)
			data := base64.StdEncoding.EncodeToString([]byte(content))
			err := bot.PostMessage(ctx, conversationId, sub.SubscriberId, bot.UuidNewV4().String(), "PLAIN_TEXT", data, config.ClientId, config.SessionId, config.PrivateKey)
			if err != nil {
				fmt.Print(err)
			}
		}
	}
}

func (impl *postsImpl) updatePost(w http.ResponseWriter, r *http.Request, params map[string]string) {
	postId := params["id"]
	if _, err := uuid.FromString(postId); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	var body models.PostBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}
	current := middlewares.CurrentUser(r)
	body.PostId = postId

	if post, err := models.UpdatePost(r.Context(), current, body); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderPost(w, r, post)
	}
}

func (impl *postsImpl) updateDraft(w http.ResponseWriter, r *http.Request, params map[string]string) {
	postId := params["id"]
	if _, err := uuid.FromString(postId); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	var body models.PostBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}
	current := middlewares.CurrentUser(r)
	body.PostId = postId

	if post, err := models.UpdateDraft(r.Context(), current, body); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		if body.IpfsId != "" {
			sendNotifyToSubscribers(r.Context(), post)
		}
		views.RenderPost(w, r, post)
	}
}

func (impl *postsImpl) deleteDraft(w http.ResponseWriter, r *http.Request, params map[string]string) {
	postId := params["id"]
	if _, err := uuid.FromString(postId); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}
	current := middlewares.CurrentUser(r)

	if err := models.DeleteDraft(r.Context(), current, params["id"]); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		result := make(map[string]bool)
		result["success"] = true
		views.RenderDataResponse(w, r, result)
	}
}

func (impl *postsImpl) getUserTelegraphPosts(w http.ResponseWriter, r *http.Request, params map[string]string) {
	limit, offset, err := getUrlParams(r)
	if err != nil {
		views.RenderErrorResponse(w, r, errors.New("Params error"))
		return
	}

	current := middlewares.CurrentUser(r)
	if list, err := models.FindTelegraphPostsByUser(r.Context(), current, offset, limit); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else if list == nil {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	} else {
		views.RenderTelegraphPosts(w, r, list)
	}
}

func (impl *postsImpl) getUserPosts(w http.ResponseWriter, r *http.Request, params map[string]string) {
	limit, offset, err := getUrlParams(r)
	if err != nil {
		views.RenderErrorResponse(w, r, errors.New("Params error"))
		return
	}
	var postType = "draft"
	if val, ok := params["type"]; ok {
		postType = val
	}

	current := middlewares.CurrentUser(r)

	if list, count, err := models.FindPostsByUser(r.Context(), current, offset, limit, postType); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else if list == nil {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	} else {
		views.RenderPosts(w, r, list, count)
	}
}

func (impl *postsImpl) getPost(w http.ResponseWriter, r *http.Request, params map[string]string) {
	postId := params["id"]
	if _, err := uuid.FromString(postId); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}
	if post, err := models.FindPostByPostId(r.Context(), postId); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else if post == nil {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	} else {
		views.RenderPost(w, r, post)
	}
}

func (impl *postsImpl) getUserDraft(w http.ResponseWriter, r *http.Request, params map[string]string) {
	postId := params["id"]
	if _, err := uuid.FromString(postId); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}
	current := middlewares.CurrentUser(r)
	if post, err := models.FindPostByPostId(r.Context(), postId); err != nil && current.UserId == post.UserId {
		views.RenderErrorResponse(w, r, err)
	} else if post == nil {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	} else {
		views.RenderPost(w, r, post)
	}
}

func (impl *postsImpl) getUserDrafts(w http.ResponseWriter, r *http.Request, params map[string]string) {
	limit, offset, err := getUrlParams(r)
	if err != nil {
		views.RenderErrorResponse(w, r, errors.New("Params error"))
		return
	}

	current := middlewares.CurrentUser(r)
	if list, count, err := models.FindDraftsByUser(r.Context(), current, offset, limit); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else if list == nil {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	} else {
		views.RenderPosts(w, r, list, count)
	}
}

func (impl *postsImpl) getAllPosts(w http.ResponseWriter, r *http.Request, params map[string]string) {
	limit, offset, err := getUrlParams(r)
	if err != nil {
		views.RenderErrorResponse(w, r, errors.New("Params error"))
	}
	if list, count, err := models.FindAllPosts(r.Context(), offset, limit); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderAllPosts(w, r, list, count)
	}
}

// TODO
func (impl *postsImpl) verifyTrace(w http.ResponseWriter, r *http.Request, params map[string]string) {
	current := middlewares.CurrentUser(r)

	if post, err := models.VerifyTrace(r.Context(), current, params["id"]); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else if post == nil {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	} else {
		views.RenderPost(w, r, post)
	}
}

func getUrlParams(r *http.Request) (int, int, error) {
	var offset, limit = 0, 20
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return 0, 0, err
	}
	if val, ok := query["offset"]; ok {
		offset, err = strconv.Atoi(val[0])
	}
	if val, ok := query["limit"]; ok {
		limit, err = strconv.Atoi(val[0])
	}
	return limit, offset, nil
}
