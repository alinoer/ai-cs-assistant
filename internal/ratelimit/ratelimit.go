package ratelimit

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter membatasi request per IP, ditambah batas global untuk semua IP.
type Limiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	perIP      rate.Limit
	burst      int
	global     *rate.Limiter
	trustProxy bool
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func New(perIP rate.Limit, burst int, global *rate.Limiter, trustProxy bool) *Limiter {
	l := &Limiter{
		visitors:   make(map[string]*visitor),
		perIP:      perIP,
		burst:      burst,
		global:     global,
		trustProxy: trustProxy,
	}
	go l.cleanup() // bersihkan IP yang sudah lama tidak aktif
	return l
}

// Middleware membungkus handler: cek jatah dulu, baru teruskan.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allowIP(l.clientIP(r)) {
			tooMany(w, "Terlalu banyak pertanyaan. Coba lagi dalam beberapa detik.")
			return
		}
		if !l.global.Allow() {
			tooMany(w, "Layanan sedang ramai. Coba lagi sebentar.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) allowIP(ip string) bool {
	l.mu.Lock()
	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.perIP, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	l.mu.Unlock()

	return v.limiter.Allow()
}

// clientIP mengambil IP pengunjung. Di belakang proxy hosting, IP asli ada di
// header X-Forwarded-For; header ini hanya dipercaya kalau TRUST_PROXY=true.
func (l *Limiter) clientIP(r *http.Request) string {
	if l.trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *Limiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

func tooMany(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "10")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

