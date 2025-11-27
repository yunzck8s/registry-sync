package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"registry-sync/internal/db/models"
	"registry-sync/internal/db/store"
	"registry-sync/pkg/config"
	"registry-sync/pkg/registry"
)

// RegistryHandler handles registry-related requests
type RegistryHandler struct {
	store *store.Store
}

// NewRegistryHandler creates a new registry handler
func NewRegistryHandler(store *store.Store) *RegistryHandler {
	return &RegistryHandler{store: store}
}

// CreateRegistry creates a new registry
// POST /api/v1/registries
func (h *RegistryHandler) CreateRegistry(c *gin.Context) {
	var req models.Registry
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Debug log
	log.Printf("DEBUG: Creating registry '%s', password length: %d, password: '%s'", req.Name, len(req.Password), req.Password)

	if err := h.store.CreateRegistry(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Clear password before sending response
	req.Password = ""

	c.JSON(http.StatusCreated, req)
}

// GetRegistry gets a registry by ID
// GET /api/v1/registries/:id
func (h *RegistryHandler) GetRegistry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	reg, err := h.store.GetRegistry(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registry not found"})
		return
	}

	// Clear password before sending response
	reg.Password = ""

	c.JSON(http.StatusOK, reg)
}

// ListRegistries lists all registries
// GET /api/v1/registries
func (h *RegistryHandler) ListRegistries(c *gin.Context) {
	regs, err := h.store.ListRegistries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Clear passwords before sending response
	for i := range regs {
		regs[i].Password = ""
	}

	c.JSON(http.StatusOK, regs)
}

// UpdateRegistry updates a registry
// PUT /api/v1/registries/:id
func (h *RegistryHandler) UpdateRegistry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	var req models.Registry
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uint(id)

	// If password is empty, preserve the existing password
	if req.Password == "" {
		existing, err := h.store.GetRegistry(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "registry not found"})
			return
		}
		req.Password = existing.Password
	}

	if err := h.store.UpdateRegistry(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Clear password before sending response
	req.Password = ""

	c.JSON(http.StatusOK, req)
}

// DeleteRegistry deletes a registry
// DELETE /api/v1/registries/:id
func (h *RegistryHandler) DeleteRegistry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	if err := h.store.DeleteRegistry(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "registry deleted"})
}

// TestRegistry tests registry connection
// POST /api/v1/registries/:id/test
func (h *RegistryHandler) TestRegistry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	reg, err := h.store.GetRegistry(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registry not found"})
		return
	}

	// Create registry client
	client := registry.NewClient(
		config.NormalizeRegistryURL(reg.URL),
		reg.Username,
		reg.Password,
		reg.Insecure,
		reg.RateLimit,
	)

	// Test connection
	ctx := context.Background()
	if err := client.PingCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "registry connection failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "registry connection test successful",
		"registry": reg.Name,
	})
}

// ListProjects lists all projects in a registry
// GET /api/v1/registries/:id/projects
func (h *RegistryHandler) ListProjects(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	reg, err := h.store.GetRegistry(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registry not found"})
		return
	}

	// Create registry client
	client := registry.NewClient(
		config.NormalizeRegistryURL(reg.URL),
		reg.Username,
		reg.Password,
		reg.Insecure,
		reg.RateLimit,
	)

	// List projects
	ctx := context.Background()
	projects, err := client.ListProjects(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list projects",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, projects)
}

// ListRepositories lists all repositories in a project
// GET /api/v1/registries/:id/projects/:project/repositories
func (h *RegistryHandler) ListRepositories(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	project := c.Param("project")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project name is required"})
		return
	}

	reg, err := h.store.GetRegistry(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registry not found"})
		return
	}

	// Create registry client
	client := registry.NewClient(
		config.NormalizeRegistryURL(reg.URL),
		reg.Username,
		reg.Password,
		reg.Insecure,
		reg.RateLimit,
	)

	// List repositories
	ctx := context.Background()
	repos, err := client.ListRepositories(ctx, project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list repositories",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, repos)
}
