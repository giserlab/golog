package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// handleForm[T] 的 T 由 handler 签名推断：handler 形如
// func(*gin.Context, *XxxRequest) 时 T 是指针类型。回归测试确保：
// 1. 指针型 T 的 GET（form 绑定）不再触发 gin validator 的
//    "reflect.Value.Interface on zero Value" panic；
// 2. 值型 T 行为不变。

type formBindingTestRequest struct {
	Title string `form:"title" conform:"trim"`
}

func TestHandleFormPointerT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var got formBindingTestRequest
	handler := handleForm(func(c *gin.Context, req *formBindingTestRequest) {
		got = *req
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x?title=%20%20Hello%20World", nil)

	// 不得 panic
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got.Title != "Hello World" {
		t.Errorf("bound Title = %q, want %q", got.Title, "Hello World")
	}
}

func TestHandleFormValueT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var got formBindingTestRequest
	handler := handleForm(func(c *gin.Context, req formBindingTestRequest) {
		got = req
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x?title=hi", nil)

	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got.Title != "hi" {
		t.Errorf("bound Title = %q, want %q", got.Title, "hi")
	}
}

// 用户报告的路径：GET /api/posts 带 query 参数。
// 无 token 时应返回 401（认证阶段），而不是 panic。
func TestAPIPostGetNoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	urls := []string{
		"/api/posts?title=Hello&type=whisper&slug=hello&visibility=public&content=Test",
		"/api/posts?%20%20%20%20title=%20Hello&%20%20%20%20type=%20whisper",
	}
	for _, u := range urls {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, u, nil)
		handleForm(APIPostGet)(c) // 不得 panic
		if w.Code != http.StatusUnauthorized {
			t.Errorf("url=%s status = %d, want 401", u, w.Code)
		}
	}
}
