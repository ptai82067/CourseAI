package main

import (
	"context"
	"courseai/backend/internal/config"
	"courseai/backend/internal/database"
	"courseai/backend/internal/handlers"
	"courseai/backend/internal/middleware"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Connect to database
	if err := database.Connect(cfg.Database.DSN()); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	if err := database.AutoMigrate(); err != nil {
		// In development, do not exit the process on migration error. Log and continue.
		if os.Getenv("ENV") == "production" {
			log.Fatal("Failed to run migrations:", err)
		}
		log.Println("Warning: migrations failed (non-fatal in dev):", err)
	}

	// Seed data (skip seeding if migrations didn't create tables)
	if err := database.SeedData(database.GetDB()); err != nil {
		if os.Getenv("ENV") == "production" {
			log.Fatal("Failed to seed data:", err)
		}
		log.Println("Warning: seeding data failed (non-fatal in dev):", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		// allow larger request bodies to support media uploads up to 64MB
		BodyLimit: 64 << 20, // 64 MB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Configure CORS with support for multiple frontend URLs (dev and production)
	corsAllowOrigins := cfg.Server.FrontendURL
	// In production, also allow requests from Vercel domains if VERCEL_URL is set
	if vercelURL := os.Getenv("VERCEL_URL"); vercelURL != "" {
		corsAllowOrigins = corsAllowOrigins + ", https://" + vercelURL + ", https://*.vercel.app"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins: corsAllowOrigins,
		// Include Cache-Control and Pragma so browser preflight allows axios default headers
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, Cache-Control, Pragma, X-Requested-With",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg)
	programHandler := handlers.NewProgramHandler()
	subcourseHandler := handlers.NewSubcourseHandler()
	lessonHandler := handlers.NewLessonHandler()
	mediaHandler := handlers.NewMediaHandler(cfg)
	seedHandler := handlers.NewSeedHandler()
	teacherHandler := handlers.NewTeacherHandler()

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	// Routes
	api := app.Group("/api")
	// allow optional token parsing on public API so handlers can apply scope-based filtering
	api.Use(authMiddleware.TokenOptional())

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)

	// Protected auth routes
	auth.Get("/me", authMiddleware.Protected(), authHandler.Me)

	// Admin routes (protected)
	admin := api.Group("/admin", authMiddleware.Protected())

	// Programs
	admin.Get("/programs", programHandler.GetAll)
	admin.Get("/programs/:id", programHandler.GetOne)
	admin.Post("/programs", programHandler.Create)
	admin.Put("/programs/:id", programHandler.Update)
	admin.Delete("/programs/:id", programHandler.Delete)

	// Subcourses
	admin.Get("/subcourses", subcourseHandler.GetAll)
	admin.Get("/subcourses/:id", subcourseHandler.GetOne)
	admin.Get("/programs/:programId/subcourses", subcourseHandler.GetByProgram)
	admin.Post("/subcourses", subcourseHandler.Create)
	admin.Put("/subcourses/:id", subcourseHandler.Update)
	admin.Delete("/subcourses/:id", subcourseHandler.Delete)

	// Lessons
	admin.Get("/lessons", lessonHandler.GetAll)
	admin.Get("/lessons/:id", lessonHandler.GetOne)
	admin.Get("/subcourses/:subcourseId/lessons", lessonHandler.GetBySubcourse)
	admin.Post("/lessons", lessonHandler.Create)
	admin.Put("/lessons/:id", lessonHandler.Update)
	admin.Delete("/lessons/:id", lessonHandler.Delete)

	// Teachers
	admin.Get("/teachers", teacherHandler.GetAll)
	admin.Post("/teachers", teacherHandler.Create)
	admin.Get("/teachers/history", teacherHandler.GetTeacherHistory)
	admin.Put("/teachers/:id/program-assignments", teacherHandler.AssignPrograms)
	admin.Put("/teachers/:id/subcourse-assignments", teacherHandler.AssignSubcourses)
	admin.Get("/teachers/:teacherId/assignments", teacherHandler.GetAssignments)
	admin.Get("/teachers/:teacherId/lesson-history", teacherHandler.GetTeacherLessonHistory)

	// Media upload
	admin.Post("/media/upload", mediaHandler.Upload)
	// Admin seed trigger (protected)
	admin.Post("/seed", seedHandler.Run)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "CourseAI Backend is running",
		})
	})

	// Public read-only routes (no auth) for frontend public pages
	// Programs - use public version that shows all programs (no access control)
	api.Get("/programs", programHandler.GetAllPublic)
	api.Get("/programs/:id", programHandler.GetOne)
	api.Get("/programs/:programId/subcourses", subcourseHandler.GetByProgram)

	// Subcourses (ensure optional token parsing is applied at route level)
	api.Get("/subcourses", authMiddleware.TokenOptional(), subcourseHandler.GetAll)
	api.Get("/subcourses/:id", authMiddleware.TokenOptional(), subcourseHandler.GetOne)

	// Lessons - use public version that shows all lessons (no access control)
	api.Get("/lessons", lessonHandler.GetAllPublic)
	api.Get("/lessons/:id", lessonHandler.GetOne)
	api.Get("/subcourses/:subcourseId/lessons", lessonHandler.GetBySubcourse)

	// Root handler: provide a friendly message at / to avoid 404s when browsing to http://localhost:PORT/
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("CourseAI Backend is running. API root: /api, health: /health")
	})

	// Serve uploads directory
	app.Static("/uploads", "./uploads")

	// Start server: bind strictly to configured port. Do NOT auto-jump ports.
	startPort, _ := strconv.Atoi(cfg.Server.Port)
	if startPort == 8082 {
		log.Fatal("Configured server port cannot be Adminer reserved port 8082")
	}
	addr := fmt.Sprintf(":%d", startPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Detect port-in-use and exit with clear message
		if strings.Contains(err.Error(), "address already in use") || strings.Contains(err.Error(), "Only one usage of each socket address") || strings.Contains(err.Error(), "bind: address") {
			log.Println("PORT ALREADY IN USE")
			os.Exit(1)
		}
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	// write chosen port to repo logs folder for scripts to pick up. Working directory is backend/, so ../logs points to repo logs.
	// Skip in production to avoid write permission issues
	if os.Getenv("ENV") != "production" {
		portFile := "../logs/backend.port"
		_ = os.WriteFile(portFile, []byte(strconv.Itoa(startPort)), 0644)

		// write PID for stop scripts
		pidFile := "../logs/backend.pid"
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	}

	log.Printf("🚀 Server starting on http://localhost%s", addr)
	log.Printf("📚 Admin credentials: admin123 / admin123")
	// start Fiber using the prepared listener in a goroutine so we can handle signals
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- app.Listener(listener)
	}()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("Received signal %s: initiating graceful shutdown...", sig.String())
		// give shutdown a deadline and wait for Shutdown to finish
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			if err := app.Shutdown(); err != nil {
				log.Println("Error during app.Shutdown():", err)
			}
			close(done)
		}()

		select {
		case <-done:
			// shutdown finished within deadline
		case <-ctx.Done():
			log.Println("Graceful shutdown timed out, forcing close")
		}

		// ensure listener closed
		_ = listener.Close()
		// remove pid file if present (only in development)
		if os.Getenv("ENV") != "production" {
			pidFile := ""
			_ = os.Remove(pidFile)
		}
		log.Println("Graceful shutdown complete")
	case err := <-serveErrCh:
		if err != nil {
			// check for bind/port errors surfaced after listener accepted
			if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") || strings.Contains(err.Error(), "Only one usage of each socket address") {
				log.Println("PORT ALREADY IN USE")
				os.Exit(1)
			}
			log.Fatal("Server error:", err)
		}
	}
}
