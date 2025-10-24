package middlewares

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct{
	mu sync.Mutex
	visitors map[string] int
	limit int
	resetTime time.Duration
}

func NewRateLimiter(limit int, resetTime time.Duration) *rateLimiter{
	rl := &rateLimiter{
		visitors: make(map[string]int),
		limit: limit,
		resetTime: resetTime,
	 }
	 go rl.resetVisterCounter()
	 return rl
}

func (rl *rateLimiter) resetVisterCounter(){
	for{
		time.Sleep(rl.resetTime)
		rl.mu.Lock()
		rl.visitors = make(map[string]int)
		rl.mu.Unlock()
	}
}


func (rl *rateLimiter) Middleware(next http.Handler) http.Handler{

	return http.HandlerFunc( func (w http.ResponseWriter, r *http.Request)  {
		rl.mu.Lock()
		defer rl.mu.Unlock()


		visitorIp := r.RemoteAddr

		rl.visitors[visitorIp]++

		fmt.Printf("vistiter: %s, count: %d\n", visitorIp, rl.visitors[visitorIp])

		if rl.visitors[visitorIp] > rl.limit{
			http.Error(w, "too many requests", http.StatusTooManyRequests )
			return 
		}


		next.ServeHTTP(w,r)
	})

}