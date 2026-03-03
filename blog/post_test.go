package blog_test

import (
	. "goblog/blog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPreviewContent(t *testing.T) {
	//long content
	post := Post{
		Content: "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum",
	}

	preview := post.PreviewContent(25)
	assert.NotEqual(t, len(preview), len(post.Content))
	assert.Equal(t, len(preview), 25)

	//long content
	preview = post.PreviewContent(len(post.Content) + 10)
	assert.Equal(t, len(preview), len(post.Content))
	assert.NotEqual(t, len(preview), 25)
}

func TestPermalink(t *testing.T) {
	currentTime := time.Now()

	// Post with default (empty) PostType should use "posts"
	post := Post{
		Slug: "test",
	}
	post.CreatedAt = currentTime
	assert.Equal(t, "/posts"+currentTime.Format("/2006/01/02/")+post.Slug, post.Permalink())

	// Post with a custom PostType
	post2 := Post{
		Slug:     "my-note",
		PostType: PostType{Slug: "notes"},
	}
	post2.CreatedAt = currentTime
	assert.Equal(t, "/notes"+currentTime.Format("/2006/01/02/")+post2.Slug, post2.Permalink())

	// Adminlink with default PostType
	assert.Equal(t, "/admin/posts"+currentTime.Format("/2006/01/02/")+post.Slug, post.Adminlink())

	// Adminlink with custom PostType
	assert.Equal(t, "/admin/notes"+currentTime.Format("/2006/01/02/")+post2.Slug, post2.Adminlink())
}
