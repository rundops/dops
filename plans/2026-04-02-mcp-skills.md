# Plan: MCP Skills — Injectable Context for AI Agents

**Spec:** v0.11.0 Feature 5
**Effort:** Half day
**Risk:** Medium — new content type touching domain, loader, and MCP

## Problem

AI agents using dops via MCP can execute runbooks but have no way to
receive domain knowledge or operational context. Skills fill this gap: a
markdown file paired with a runbook.yaml (`type: skill`) that gets exposed
as an MCP prompt.

## TDD Steps (sequential phases, max 5 files each)

### Phase 1: Domain + Loader (3 files)

#### Red

**File:** `internal/domain/runbook_test.go`

1. Test `TestRunbookType_DefaultIsRunbook` — Runbook with empty Type field.
   Verify `IsSkill()` returns false.

2. Test `TestRunbookType_SkillIdentified` — Runbook with `Type: "skill"`.
   Verify `IsSkill()` returns true.

**File:** `internal/catalog/loader_test.go`

3. Test `TestLoader_LoadsSkillsFromDisk` — catalog directory with a skill
   subdirectory containing `runbook.yaml` (type: skill) + `skill.md`.
   Verify skill appears in `CatalogWithRunbooks.Skills` slice.

4. Test `TestLoader_SkillWithoutMarkdown_Skipped` — skill directory with
   `runbook.yaml` but no `skill.md`. Verify warning logged, skill skipped.

5. Test `TestLoader_SkillsNotInRunbooks` — skill entry does NOT appear
   in `CatalogWithRunbooks.Runbooks`.

6. Test `TestLoader_MixedRunbooksAndSkills` — catalog with both. Verify
   runbooks in `.Runbooks`, skills in `.Skills`, correct counts.

#### Green

**File:** `internal/domain/runbook.go`

Add to Runbook struct:
```go
Type    string `yaml:"type,omitempty" json:"type,omitempty"` // "runbook" (default) or "skill"
Trigger string `yaml:"trigger,omitempty" json:"trigger,omitempty"` // comma-separated keywords
```

Add method:
```go
func (r Runbook) IsSkill() bool { return r.Type == "skill" }
```

**File:** `internal/domain/skill.go` (new)

```go
type Skill struct {
    ID          string
    Name        string
    Description string
    Trigger     string
    Content     string // raw skill.md markdown
    Catalog     string // parent catalog name
}
```

**File:** `internal/catalog/loader.go`

Add `Skills []domain.Skill` to `CatalogWithRunbooks`.

In `loadCatalog()`, after loading `runbook.yaml`:
- If `rb.IsSkill()`: read `skill.md` from same directory, create
  `domain.Skill`, append to `skills` slice. Skip adding to `runbooks`.
- If `skill.md` missing for a skill type: log warning, skip.
- If `rb.Type` is empty or "runbook": existing behavior (load script).

### Phase 2: MCP Integration (2 files)

#### Red

**File:** `internal/mcp/server_test.go`

7. Test `TestMCPServer_SkillsAsPrompts` — create server with catalogs
   containing skills. Verify skills appear in `ListPrompts`.

8. Test `TestMCPServer_GetPrompt_ReturnsSkillContent` — request prompt
   by skill ID. Verify full skill.md content returned.

9. Test `TestMCPServer_SkillsNotInTools` — verify skills do NOT appear
   in `ListTools`.

10. Test `TestMCPServer_SkillPromptMetadata` — verify prompt description
    and trigger keywords appear in prompt metadata.

#### Green

**File:** `internal/mcp/server.go`

In `registerTools()`: skip entries where `rb.IsSkill()` (they should
already be filtered out since skills are in `.Skills` not `.Runbooks`,
but add a guard).

Add `registerSkillPrompts()` method called from `NewServer()`:
```go
func (s *Server) registerSkillPrompts() {
    for _, c := range s.catalogs {
        for _, sk := range c.Skills {
            s.srv.AddPrompt(...)  // name=sk.ID, description=sk.Description
        }
    }
}
```

**File:** `internal/mcp/prompts.go`

Add handler for skill prompts that returns `sk.Content` as a text message.

### Phase 3: Verify end-to-end (0 new files)

- `go test ./... -v`
- `go build ./...`
- `go vet ./...`
- Manual: create a test catalog with a skill, run `dops mcp tools`
  (skill should NOT appear), verify via MCP prompt listing.

## Key Constraints

- Skills are NOT shown in TUI sidebar
- Skills are NOT exposed as MCP tools
- Skills have no risk level (not filtered by policy)
- `type` field defaults to "runbook" — fully backward compatible
- Skill content is the raw markdown, not processed

## Verification

```
go test ./internal/domain/... -v
go test ./internal/catalog/... -v
go test ./internal/mcp/... -v
go test ./... -v
go build ./...
go vet ./...
```
