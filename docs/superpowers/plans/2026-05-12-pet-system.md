# Pet System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a fox pet system with nurturing mechanics (feed/play) and AI-powered pet chat, accessible via corner presence, expandable panel, and `/pet` detail page.

**Architecture:** New `pet` + `pet_interaction` tables. Backend CRUD following existing handler→service→repository→model pattern. Frontend uses Pinia store for pet state, composable for interactions. Integrates with AI Chat pet mode.

**Tech Stack:** Go + Gin + MySQL (backend), Vue 3 + Naive UI + Pinia (frontend)

---

## File Structure

### Backend

| File | Change Type | Responsibility |
|------|-------------|----------------|
| `internal/model/pet.go` | Create | Pet + PetInteraction models |
| `internal/repository/pet.go` | Create | Pet CRUD + interaction log |
| `internal/service/pet.go` | Create | Pet business logic (feed, play, decay, level up) |
| `internal/handler/pet.go` | Create | Pet API handlers |
| `internal/router/router.go` | Modify | Register /pet routes |
| `internal/pkg/dbmigrate/migrations/000035_create_pets.up.sql` | Create | Migration |
| `internal/pkg/dbmigrate/migrations/000035_create_pets.down.sql` | Create | Rollback |

### Frontend

| File | Change Type | Responsibility |
|------|-------------|----------------|
| `web/src/api/index.ts` | Modify | Add `petApi` |
| `web/src/types/index.ts` | Modify | Add Pet, PetInteraction types |
| `web/src/stores/pet.ts` | Create | Pet state management |
| `web/src/components/pet/PetCorner.vue` | Create | Corner mini-display |
| `web/src/components/pet/PetPanel.vue` | Create | Expandable interaction panel |
| `web/src/pages/pet/Index.vue` | Create | Full pet detail page |
| `web/src/router/index.ts` | Modify | Add /pet route |
| `web/src/layouts/AppShell.vue` | Modify | Mount PetCorner |
| `web/src/i18n/zh-CN.ts` | Modify | Pet i18n keys |
| `web/src/i18n/en.ts` | Modify | Pet i18n keys |

---

## Backend Tasks

### Task 1: Pet Models + Migration

**Files:**
- Create: `internal/model/pet.go`
- Create: `internal/pkg/dbmigrate/migrations/000035_create_pets.up.sql`
- Create: `internal/pkg/dbmigrate/migrations/000035_create_pets.down.sql`

- [ ] **Step 1: Create Pet models**

```go
package model

// Pet represents a user's virtual pet.
type Pet struct {
	BaseModel
	UserID uint   `json:"user_id" gorm:"uniqueIndex;not null"`
	Name   string `json:"name" gorm:"size:50;not null;default:'小狐'"`
	Species string `json:"species" gorm:"size:20;not null;default:'fox'"`
	Level  int    `json:"level" gorm:"not null;default:1"`
	Exp    int    `json:"exp" gorm:"not null;default:0"`
	Hunger int    `json:"hunger" gorm:"not null;default:30"` // 0=full, 100=starving
	Mood   int    `json:"mood" gorm:"not null;default:70"`   // 0=sad, 100=happy
}

// PetInteraction records a single interaction with a pet.
type PetInteraction struct {
	BaseModel
	PetID uint   `json:"pet_id" gorm:"index;not null"`
	Type  string `json:"type" gorm:"size:20;not null"` // feed, play, chat, level_up
	Value int    `json:"value" gorm:"not null;default:0"`
}
```

- [ ] **Step 2: Create migration up**

`000035_create_pets.up.sql`:

```sql
CREATE TABLE pets (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL UNIQUE,
  name VARCHAR(50) NOT NULL DEFAULT '小狐',
  species VARCHAR(20) NOT NULL DEFAULT 'fox',
  level INT NOT NULL DEFAULT 1,
  exp INT NOT NULL DEFAULT 0,
  hunger INT NOT NULL DEFAULT 30,
  mood INT NOT NULL DEFAULT 70,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  INDEX idx_pets_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE pet_interactions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  pet_id BIGINT UNSIGNED NOT NULL,
  type VARCHAR(20) NOT NULL,
  value INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  INDEX idx_pet_interactions_pet_id (pet_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

- [ ] **Step 3: Create migration down**

`000035_create_pets.down.sql`:

```sql
DROP TABLE IF EXISTS pet_interactions;
DROP TABLE IF EXISTS pets;
```

- [ ] **Step 4: Commit**

```bash
git add internal/model/pet.go internal/pkg/dbmigrate/migrations/000035_create_pets.* && git commit -m "feat(pet): pet + pet_interaction models + migration"
```

---

### Task 2: Pet Repository

**Files:**
- Create: `internal/repository/pet.go`

- [ ] **Step 1: Create repository**

```go
package repository

import (
	"context"
	"github.com/sreagent/sreagent/internal/model"
	"gorm.io/gorm"
)

type PetRepository struct {
	db *gorm.DB
}

func NewPetRepository(db *gorm.DB) *PetRepository {
	return &PetRepository{db: db}
}

func (r *PetRepository) GetByUserID(ctx context.Context, userID uint) (*model.Pet, error) {
	var pet model.Pet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&pet).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

func (r *PetRepository) Create(ctx context.Context, pet *model.Pet) error {
	return r.db.WithContext(ctx).Create(pet).Error
}

func (r *PetRepository) Update(ctx context.Context, pet *model.Pet) error {
	return r.db.WithContext(ctx).Save(pet).Error
}

func (r *PetRepository) CreateInteraction(ctx context.Context, interaction *model.PetInteraction) error {
	return r.db.WithContext(ctx).Create(interaction).Error
}

func (r *PetRepository) ListInteractions(ctx context.Context, petID uint, limit int) ([]model.PetInteraction, error) {
	var interactions []model.PetInteraction
	err := r.db.WithContext(ctx).
		Where("pet_id = ?", petID).
		Order("created_at DESC").
		Limit(limit).
		Find(&interactions).Error
	return interactions, err
}

// ApplyDecay increases hunger and decreases mood for all pets.
// Called periodically (e.g., via cron or on access).
func (r *PetRepository) ApplyDecay(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&model.Pet{}).
		Where("hunger < 100").
		Update("hunger", gorm.Expr("LEAST(hunger + 1, 100)")).Error
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/repository/pet.go && git commit -m "feat(pet): pet repository"
```

---

### Task 3: Pet Service

**Files:**
- Create: `internal/service/pet.go`

- [ ] **Step 1: Create service**

```go
package service

import (
	"context"
	"fmt"
	"math"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

type PetService struct {
	repo *repository.PetRepository
}

func NewPetService(repo *repository.PetRepository) *PetService {
	return &PetService{repo: repo}
}

// GetOrCreate returns the user's pet, creating one if it doesn't exist.
func (s *PetService) GetOrCreate(ctx context.Context, userID uint) (*model.Pet, error) {
	pet, err := s.repo.GetByUserID(ctx, userID)
	if err == nil {
		return pet, nil
	}
	// Create new pet
	pet = &model.Pet{
		UserID:  userID,
		Name:    "小狐",
		Species: "fox",
		Level:   1,
		Exp:     0,
		Hunger:  30,
		Mood:    70,
	}
	if err := s.repo.Create(ctx, pet); err != nil {
		return nil, fmt.Errorf("failed to create pet: %w", err)
	}
	return pet, nil
}

// Update updates pet info (name, etc.).
func (s *PetService) Update(ctx context.Context, pet *model.Pet) error {
	return s.repo.Update(ctx, pet)
}

// Feed reduces hunger by 20 and grants 5 exp.
func (s *PetService) Feed(ctx context.Context, userID uint) (*model.Pet, error) {
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}

	pet.Hunger = max(0, pet.Hunger-20)
	pet.Exp += 5

	s.checkLevelUp(pet)

	if err := s.repo.Update(ctx, pet); err != nil {
		return nil, err
	}

	_ = s.repo.CreateInteraction(ctx, &model.PetInteraction{
		PetID: pet.ID,
		Type:  "feed",
		Value: 20,
	})

	return pet, nil
}

// Play increases mood by 15 and grants 5 exp.
func (s *PetService) Play(ctx context.Context, userID uint) (*model.Pet, error) {
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}

	pet.Mood = min(100, pet.Mood+15)
	pet.Exp += 5

	s.checkLevelUp(pet)

	if err := s.repo.Update(ctx, pet); err != nil {
		return nil, err
	}

	_ = s.repo.CreateInteraction(ctx, &model.PetInteraction{
		PetID: pet.ID,
		Type:  "play",
		Value: 15,
	})

	return pet, nil
}

// AddChatExp grants exp for chatting with the pet.
func (s *PetService) AddChatExp(ctx context.Context, userID uint) {
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return
	}
	pet.Exp += 2
	s.checkLevelUp(pet)
	_ = s.repo.Update(ctx, pet)
	_ = s.repo.CreateInteraction(ctx, &model.PetInteraction{
		PetID: pet.ID,
		Type:  "chat",
		Value: 2,
	})
}

// GetInteractions returns recent interactions.
func (s *PetService) GetInteractions(ctx context.Context, userID uint, limit int) ([]model.PetInteraction, error) {
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListInteractions(ctx, pet.ID, limit)
}

// checkLevelUp checks if the pet has enough exp to level up.
// Formula: required = level * 100
func (s *PetService) checkLevelUp(pet *model.Pet) {
	required := pet.Level * 100
	for pet.Exp >= required {
		pet.Exp -= required
		pet.Level++
		required = pet.Level * 100
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/pet.go && git commit -m "feat(pet): pet service with feed/play/level-up"
```

---

### Task 4: Pet Handler + Router

**Files:**
- Create: `internal/handler/pet.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Create handler**

```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sreagent/sreagent/internal/service"
)

type PetHandler struct {
	petSvc *service.PetService
}

func NewPetHandler(petSvc *service.PetService) *PetHandler {
	return &PetHandler{petSvc: petSvc}
}

// GetPet handles GET /pet
func (h *PetHandler) GetPet(c *gin.Context) {
	userID, _ := h.GetCurrentUserID(c)
	pet, err := h.petSvc.GetOrCreate(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// UpdatePet handles PUT /pet
func (h *PetHandler) UpdatePet(c *gin.Context) {
	userID, _ := h.GetCurrentUserID(c)
	pet, err := h.petSvc.GetOrCreate(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}

	if req.Name != "" {
		pet.Name = req.Name
	}
	if err := h.petSvc.Update(c.Request.Context(), pet); err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// FeedPet handles POST /pet/feed
func (h *PetHandler) FeedPet(c *gin.Context) {
	userID, _ := h.GetCurrentUserID(c)
	pet, err := h.petSvc.Feed(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// PlayWithPet handles POST /pet/play
func (h *PetHandler) PlayWithPet(c *gin.Context) {
	userID, _ := h.GetCurrentUserID(c)
	pet, err := h.petSvc.Play(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// GetInteractions handles GET /pet/interactions
func (h *PetHandler) GetInteractions(c *gin.Context) {
	userID, _ := h.GetCurrentUserID(c)
	interactions, err := h.petSvc.GetInteractions(c.Request.Context(), userID, 50)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, interactions)
}
```

- [ ] **Step 2: Register routes**

Add `Pet *handler.PetHandler` to the `Handlers` struct in `router.go`.

Add routes after the AI group:

```go
// Pet — all authenticated users
pet := auth.Group("/pet")
{
	pet.GET("", handlers.Pet.GetPet)
	pet.PUT("", handlers.Pet.UpdatePet)
	pet.POST("/feed", handlers.Pet.FeedPet)
	pet.POST("/play", handlers.Pet.PlayWithPet)
	pet.GET("/interactions", handlers.Pet.GetInteractions)
}
```

- [ ] **Step 3: Update DI wiring**

In `cmd/server/main.go`, in the repository section (around line 137), add:

```go
petRepo := repository.NewPetRepository(db)
```

In the service section (around line 193), add:

```go
petSvc := service.NewPetService(petRepo)
```

In the `router.Handlers` struct (around line 489), add:

```go
Pet: handler.NewPetHandler(petSvc),
```

- [ ] **Step 4: Add Pet models to autoMigrate**

In `cmd/server/main.go` `autoMigrate` function (around line 666), add:

```go
// Pet system
models = append(models, &model.Pet{}, &model.PetInteraction{})
```

- [ ] **Step 5: Commit**

```bash
git add internal/handler/pet.go internal/router/router.go cmd/server/main.go && git commit -m "feat(pet): pet handler + router + DI wiring"
```

---

## Frontend Tasks

### Task 5: Pet Types + API

**Files:**
- Modify: `web/src/types/index.ts`
- Modify: `web/src/api/index.ts`

- [ ] **Step 1: Add Pet types**

In `types/index.ts`:

```typescript
export interface Pet {
  id: number
  user_id: number
  name: string
  species: string
  level: number
  exp: number
  hunger: number
  mood: number
  created_at: string
  updated_at: string
}

export interface PetInteraction {
  id: number
  pet_id: number
  type: 'feed' | 'play' | 'chat' | 'level_up'
  value: number
  created_at: string
}
```

- [ ] **Step 2: Add petApi**

In `api/index.ts`:

```typescript
// ===== Pet API =====
export const petApi = {
  get: () =>
    request.get<ApiResponse<Pet>>('/pet'),

  update: (data: { name?: string }) =>
    request.put<ApiResponse<Pet>>('/pet', data),

  feed: () =>
    request.post<ApiResponse<Pet>>('/pet/feed'),

  play: () =>
    request.post<ApiResponse<Pet>>('/pet/play'),

  getInteractions: () =>
    request.get<ApiResponse<PetInteraction[]>>('/pet/interactions'),
}
```

- [ ] **Step 3: Commit**

```bash
cd web && git add src/types/index.ts src/api/index.ts && git commit -m "feat(pet): pet types + API client"
```

---

### Task 6: Pet Store

**Files:**
- Create: `web/src/stores/pet.ts`

- [ ] **Step 1: Create Pinia store**

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { petApi } from '@/api'
import type { Pet, PetInteraction } from '@/types'

export const usePetStore = defineStore('pet', () => {
  const pet = ref<Pet | null>(null)
  const interactions = ref<PetInteraction[]>([])
  const loading = ref(false)

  const expForNextLevel = computed(() => (pet.value?.level ?? 1) * 100)
  const expProgress = computed(() => {
    if (!pet.value) return 0
    return (pet.value.exp / expForNextLevel.value) * 100
  })

  async function fetchPet() {
    loading.value = true
    try {
      const { data } = await petApi.get()
      pet.value = data
    } finally {
      loading.value = false
    }
  }

  async function updateName(name: string) {
    if (!pet.value) return
    const { data } = await petApi.update({ name })
    pet.value = data
  }

  async function feed() {
    const { data } = await petApi.feed()
    pet.value = data
  }

  async function play() {
    const { data } = await petApi.play()
    pet.value = data
  }

  async function fetchInteractions() {
    const { data } = await petApi.getInteractions()
    interactions.value = data || []
  }

  return {
    pet,
    interactions,
    loading,
    expForNextLevel,
    expProgress,
    fetchPet,
    updateName,
    feed,
    play,
    fetchInteractions,
  }
})
```

- [ ] **Step 2: Commit**

```bash
cd web && git add src/stores/pet.ts && git commit -m "feat(pet): pet Pinia store"
```

---

### Task 7: PetCorner Component

**Files:**
- Create: `web/src/components/pet/PetCorner.vue`

- [ ] **Step 1: Create corner mini-display**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NIcon, NPopover, NProgress } from 'naive-ui'
import { usePetStore } from '@/stores/pet'
import PetPanel from './PetPanel.vue'

const petStore = usePetStore()
const showPanel = ref(false)

onMounted(() => {
  petStore.fetchPet()
})

function getStatusEmoji() {
  if (!petStore.pet) return '🦊'
  if (petStore.pet.hunger > 80) return '😫'
  if (petStore.pet.mood < 30) return '😢'
  if (petStore.pet.mood > 80) return '😊'
  return '🦊'
}
</script>

<template>
  <div v-if="petStore.pet" class="pet-corner" @click="showPanel = !showPanel">
    <div class="pet-corner-avatar">
      <span class="pet-corner-emoji">{{ getStatusEmoji() }}</span>
    </div>
    <div class="pet-corner-info">
      <div class="pet-corner-name">{{ petStore.pet.name }}</div>
      <div class="pet-corner-level">Lv.{{ petStore.pet.level }}</div>
    </div>

    <n-popover
      :show="showPanel"
      placement="top-end"
      :show-arrow="false"
      trigger="manual"
      @update:show="showPanel = $event"
    >
      <template #trigger>
        <div />
      </template>
      <PetPanel @close="showPanel = false" />
    </n-popover>
  </div>
</template>

<style scoped>
.pet-corner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: var(--sre-radius-md);
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.pet-corner:hover {
  background: var(--sre-bg-hover);
}

.pet-corner-avatar {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.pet-corner-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.pet-corner-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-primary);
  line-height: 1.2;
}

.pet-corner-level {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  line-height: 1.2;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
cd web && git add src/components/pet/PetCorner.vue && git commit -m "feat(pet): PetCorner mini-display"
```

---

### Task 8: PetPanel Component

**Files:**
- Create: `web/src/components/pet/PetPanel.vue`

- [ ] **Step 1: Create expandable panel**

```vue
<script setup lang="ts">
import { NButton, NIcon, NProgress } from 'naive-ui'
import { HeartOutline, HappyOutline, RestaurantOutline, ChatbubbleEllipsesOutline } from '@vicons/ionicons5'
import { useRouter } from 'vue-router'
import { usePetStore } from '@/stores/pet'
import { useI18n } from 'vue-i18n'

const emit = defineEmits<{
  close: []
}>()

const router = useRouter()
const petStore = usePetStore()
const { t } = useI18n()

async function handleFeed() {
  await petStore.feed()
}

async function handlePlay() {
  await petStore.play()
}

function handleChat() {
  emit('close')
  // Open AI chat in pet mode — handled by parent
  // For now, navigate to pet page
  router.push('/pet')
}

function handleDetail() {
  emit('close')
  router.push('/pet')
}
</script>

<template>
  <div v-if="petStore.pet" class="pet-panel">
    <div class="pet-panel-header">
      <span class="pet-panel-name">{{ petStore.pet.name }}</span>
      <span class="pet-panel-level">Lv.{{ petStore.pet.level }}</span>
    </div>

    <!-- Status bars -->
    <div class="pet-panel-stats">
      <div class="pet-stat">
        <span class="pet-stat-label">{{ t('pet.hunger') }}</span>
        <n-progress
          :percentage="100 - petStore.pet.hunger"
          :show-indicator="false"
          :height="6"
          :color="petStore.pet.hunger > 70 ? 'var(--sre-critical)' : 'var(--sre-warning)'"
          rail-color="var(--sre-bg-sunken)"
        />
      </div>
      <div class="pet-stat">
        <span class="pet-stat-label">{{ t('pet.mood') }}</span>
        <n-progress
          :percentage="petStore.pet.mood"
          :show-indicator="false"
          :height="6"
          :color="petStore.pet.mood < 30 ? 'var(--sre-critical)' : 'var(--sre-success)'"
          rail-color="var(--sre-bg-sunken)"
        />
      </div>
      <div class="pet-stat">
        <span class="pet-stat-label">EXP</span>
        <n-progress
          :percentage="petStore.expProgress"
          :show-indicator="false"
          :height="6"
          color="var(--sre-primary)"
          rail-color="var(--sre-bg-sunken)"
        />
        <span class="pet-stat-exp">{{ petStore.pet.exp }}/{{ petStore.expForNextLevel }}</span>
      </div>
    </div>

    <!-- Actions -->
    <div class="pet-panel-actions">
      <n-button size="small" @click="handleFeed">
        <template #icon><n-icon :component="RestaurantOutline" /></template>
        {{ t('pet.feed') }}
      </n-button>
      <n-button size="small" @click="handlePlay">
        <template #icon><n-icon :component="HappyOutline" /></template>
        {{ t('pet.play') }}
      </n-button>
      <n-button size="small" @click="handleChat">
        <template #icon><n-icon :component="ChatbubbleEllipsesOutline" /></template>
        {{ t('pet.chat') }}
      </n-button>
    </div>

    <div class="pet-panel-link" @click="handleDetail">
      {{ t('pet.viewDetail') }} →
    </div>
  </div>
</template>

<style scoped>
.pet-panel {
  width: 240px;
  padding: 12px;
}

.pet-panel-header {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 12px;
}

.pet-panel-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.pet-panel-level {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.pet-panel-stats {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.pet-stat {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pet-stat-label {
  font-size: 11px;
  color: var(--sre-text-secondary);
  width: 32px;
  flex-shrink: 0;
}

.pet-stat-exp {
  font-size: 10px;
  color: var(--sre-text-tertiary);
  white-space: nowrap;
}

.pet-panel-actions {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}

.pet-panel-link {
  font-size: 12px;
  color: var(--sre-primary);
  cursor: pointer;
  text-align: center;
}

.pet-panel-link:hover {
  text-decoration: underline;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
cd web && git add src/components/pet/PetPanel.vue && git commit -m "feat(pet): PetPanel with feed/play/chat actions"
```

---

### Task 9: Pet Detail Page + Route

**Files:**
- Create: `web/src/pages/pet/Index.vue`
- Modify: `web/src/router/index.ts`

- [ ] **Step 1: Create pet detail page**

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { NCard, NButton, NIcon, NProgress, NDataTable, NInput } from 'naive-ui'
import { RestaurantOutline, HappyOutline, ChatbubbleEllipsesOutline } from '@vicons/ionicons5'
import { usePetStore } from '@/stores/pet'
import { useI18n } from 'vue-i18n'
import type { DataTableColumns } from 'naive-ui'
import type { PetInteraction } from '@/types'

const petStore = usePetStore()
const { t } = useI18n()

onMounted(async () => {
  await petStore.fetchPet()
  await petStore.fetchInteractions()
})

const columns: DataTableColumns<PetInteraction> = [
  { title: t('pet.interactionType'), key: 'type', width: 100 },
  { title: t('pet.interactionValue'), key: 'value', width: 80 },
  { title: t('pet.interactionTime'), key: 'created_at', width: 180 },
]

function getTypeLabel(type: string) {
  const map: Record<string, string> = {
    feed: t('pet.feed'),
    play: t('pet.play'),
    chat: t('pet.chat'),
    level_up: t('pet.levelUp'),
  }
  return map[type] || type
}
</script>

<template>
  <div class="pet-page" v-if="petStore.pet">
    <div class="pet-page-header">
      <h1>{{ petStore.pet.name }}</h1>
      <span class="pet-page-level">Lv.{{ petStore.pet.level }}</span>
    </div>

    <div class="pet-page-grid">
      <!-- Pet avatar + status -->
      <n-card class="pet-card">
        <div class="pet-avatar-area">
          <span class="pet-avatar-emoji">🦊</span>
        </div>
        <div class="pet-status-bars">
          <div class="pet-bar">
            <span>{{ t('pet.hunger') }}</span>
            <n-progress :percentage="100 - petStore.pet.hunger" :height="8" />
          </div>
          <div class="pet-bar">
            <span>{{ t('pet.mood') }}</span>
            <n-progress :percentage="petStore.pet.mood" :height="8" />
          </div>
          <div class="pet-bar">
            <span>EXP</span>
            <n-progress :percentage="petStore.expProgress" :height="8" />
            <span class="pet-exp-text">{{ petStore.pet.exp }}/{{ petStore.expForNextLevel }}</span>
          </div>
        </div>
        <div class="pet-actions">
          <n-button @click="petStore.feed()">
            <template #icon><n-icon :component="RestaurantOutline" /></template>
            {{ t('pet.feed') }}
          </n-button>
          <n-button @click="petStore.play()">
            <template #icon><n-icon :component="HappyOutline" /></template>
            {{ t('pet.play') }}
          </n-button>
        </div>
      </n-card>

      <!-- Pet settings -->
      <n-card :title="t('pet.settings')">
        <div class="pet-setting">
          <label>{{ t('pet.name') }}</label>
          <n-input
            :value="petStore.pet.name"
            @update:value="petStore.updateName($event)"
            :placeholder="t('pet.namePlaceholder')"
          />
        </div>
      </n-card>
    </div>

    <!-- Interaction history -->
    <n-card :title="t('pet.interactionHistory')" class="pet-history-card">
      <n-data-table
        :columns="columns"
        :data="petStore.interactions"
        :bordered="false"
        size="small"
      />
    </n-card>
  </div>
</template>

<style scoped>
.pet-page {
  padding: 24px;
  max-width: 800px;
}

.pet-page-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 24px;
}

.pet-page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
}

.pet-page-level {
  font-size: 14px;
  color: var(--sre-text-tertiary);
}

.pet-page-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

.pet-avatar-area {
  text-align: center;
  font-size: 64px;
  margin-bottom: 16px;
}

.pet-status-bars {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.pet-bar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pet-bar span:first-child {
  font-size: 12px;
  color: var(--sre-text-secondary);
  width: 32px;
}

.pet-exp-text {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  white-space: nowrap;
}

.pet-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.pet-setting {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pet-setting label {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-secondary);
}

.pet-history-card {
  margin-top: 20px;
}
</style>
```

- [ ] **Step 2: Add route**

In `router/index.ts`, add under Platform section or as a top-level route:

```typescript
{ path: 'pet', component: () => import('@/pages/pet/Index.vue'), meta: { title: '我的宠物' } },
```

- [ ] **Step 3: Commit**

```bash
cd web && git add src/pages/pet/Index.vue src/router/index.ts && git commit -m "feat(pet): pet detail page + route"
```

---

### Task 10: AppShell Integration + i18n

**Files:**
- Modify: `web/src/layouts/AppShell.vue`
- Modify: `web/src/i18n/zh-CN.ts`
- Modify: `web/src/i18n/en.ts`

- [ ] **Step 1: Mount PetCorner in AppShell**

Import and place `PetCorner` in the sidebar bottom area or the main content area's bottom-right corner.

```typescript
import PetCorner from '@/components/pet/PetCorner.vue'
```

Place in template (in the rail-bottom or sidebar-bottom area):

```html
<PetCorner />
```

- [ ] **Step 2: Add pet i18n keys**

In `zh-CN.ts`:

```typescript
pet: {
  hunger: '饥饿',
  mood: '心情',
  feed: '喂食',
  play: '玩耍',
  chat: '对话',
  viewDetail: '查看详情',
  settings: '宠物设置',
  name: '名字',
  namePlaceholder: '给你的宠物起个名字',
  interactionHistory: '互动记录',
  interactionType: '类型',
  interactionValue: '数值',
  interactionTime: '时间',
  levelUp: '升级',
}
```

In `en.ts`:

```typescript
pet: {
  hunger: 'Hunger',
  mood: 'Mood',
  feed: 'Feed',
  play: 'Play',
  chat: 'Chat',
  viewDetail: 'View Details',
  settings: 'Pet Settings',
  name: 'Name',
  namePlaceholder: 'Give your pet a name',
  interactionHistory: 'Interaction History',
  interactionType: 'Type',
  interactionValue: 'Value',
  interactionTime: 'Time',
  levelUp: 'Level Up',
}
```

- [ ] **Step 3: Commit**

```bash
cd web && git add src/layouts/AppShell.vue src/i18n/zh-CN.ts src/i18n/en.ts && git commit -m "feat(pet): AppShell PetCorner + i18n keys"
```

---

## Verification

1. `go build ./...` — passes
2. `go test ./...` — passes
3. `cd web && node_modules/.bin/vue-tsc --noEmit` — passes
4. `cd web && npx vite build` — passes
5. Browser check:
   - Pet corner visible in sidebar bottom area
   - Click opens panel with hunger/mood/EXP bars
   - Feed button reduces hunger, grants EXP
   - Play button increases mood, grants EXP
   - Level up triggers when EXP reaches threshold
   - `/pet` page shows full detail with interaction history
   - Pet name is editable
