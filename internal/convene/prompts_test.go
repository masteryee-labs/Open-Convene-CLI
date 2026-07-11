// prompts_test.go — tests for BuildSynthesisPrompt and BuildExecPrompt (S6).
//
// These are pure-string functions with no external dependencies, so tests
// verify content containment, structure, and fallback behavior.

package convene

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// BuildSynthesisPrompt tests
// ---------------------------------------------------------------------------

func TestBuildSynthesisPromptContainsTask(t *testing.T) {
	task := "Write a function to sort an array"
	responses := map[string]string{
		"agy":  "Use quicksort for O(n log n)",
		"grok": "Use mergesort for stability",
	}

	prompt := BuildSynthesisPrompt(task, responses)

	assert.Contains(t, prompt, task, "synthesis prompt must contain the original task")
}

func TestBuildSynthesisPromptContainsAllResponses(t *testing.T) {
	responses := map[string]string{
		"agy":   "Response from agy model about sorting",
		"grok":  "Response from grok model about sorting",
		"codex": "Response from codex model about sorting",
	}

	prompt := BuildSynthesisPrompt("sort task", responses)

	// Every response text should appear in the prompt.
	for _, resp := range responses {
		assert.Contains(t, prompt, resp, "synthesis prompt must contain every response")
	}
}

func TestBuildSynthesisPromptUsesAnonymousLabels(t *testing.T) {
	responses := map[string]string{
		"agy":  "agy response content",
		"grok": "grok response content",
	}

	prompt := BuildSynthesisPrompt("task", responses)

	// Model names should NOT appear as labels — responses are anonymized.
	assert.NotContains(t, prompt, "Response from agy",
		"model names must NOT appear in synthesis prompt (anonymized)")
	assert.NotContains(t, prompt, "Response from grok",
		"model names must NOT appear in synthesis prompt (anonymized)")
	// Anonymous labels (A, B) should be present instead.
	assert.Contains(t, prompt, "Response A",
		"anonymous label 'A' should be present")
	assert.Contains(t, prompt, "Response B",
		"anonymous label 'B' should be present")
}

func TestBuildSynthesisPromptIncludesNoVoteInstruction(t *testing.T) {
	responses := map[string]string{
		"agy":  "response A",
		"grok": "response B",
	}

	prompt := BuildSynthesisPrompt("task", responses)

	// The prompt must instruct the synthesizer NOT to majority-vote.
	assert.Contains(t, prompt, "NOT majority-vote", "should instruct no majority voting")
	// And NOT to average.
	assert.Contains(t, prompt, "NOT average", "should instruct no averaging")
	// And TO perform reasoning-based integration.
	assert.Contains(t, prompt, "reasoning-based integration",
		"should instruct reasoning-based integration")
}

func TestBuildSynthesisPromptDeterministicOrdering(t *testing.T) {
	// The responses section should be sorted by model name (internally) for
	// determinism, but labeled anonymously (A, B, C, ...).
	// Sorted order: agy(A) < codex(B) < grok(C).
	responses := map[string]string{
		"grok":  "grok response text",
		"agy":   "agy response text",
		"codex": "codex response text",
	}

	prompt := BuildSynthesisPrompt("task", responses)

	// Find the positions of each anonymous label.
	aPos := strings.Index(prompt, "Response A")
	bPos := strings.Index(prompt, "Response B")
	cPos := strings.Index(prompt, "Response C")

	require.True(t, aPos >= 0 && bPos >= 0 && cPos >= 0,
		"all anonymous labels should be present")

	// A should appear before B, B before C (sorted by model name internally).
	assert.Less(t, aPos, bPos, "Response A should appear before Response B")
	assert.Less(t, bPos, cPos, "Response B should appear before Response C")

	// Verify content mapping: agy→A, codex→B, grok→C (alphabetical model order).
	aSection := prompt[aPos:]
	assert.Contains(t, aSection, "agy response text",
		"Response A should contain agy's text (first alphabetically)")
}

func TestBuildSynthesisPromptEmptyResponses(t *testing.T) {
	prompt := BuildSynthesisPrompt("task", map[string]string{})
	assert.Contains(t, prompt, "task", "prompt should still contain the task")
	// Should not panic with empty responses.
	assert.NotEmpty(t, prompt)
}

// ---------------------------------------------------------------------------
// BuildExecPrompt tests
// ---------------------------------------------------------------------------

func TestBuildExecPromptCodeMode(t *testing.T) {
	task := "implement a REST API endpoint"
	synthesis := "Use goroutines for concurrency"
	responses := map[string]string{"agy": "response A"}

	prompt := BuildExecPrompt(task, &synthesis, responses, "code")

	// Should contain the task.
	assert.Contains(t, prompt, task)
	// Should contain the synthesis as integrated guidance.
	assert.Contains(t, prompt, synthesis)
	// Code mode template should mention implementation/code.
	assert.Contains(t, prompt, "code", "code mode prompt should mention code/implementation")
}

func TestBuildExecPromptAgentMode(t *testing.T) {
	task := "research and deploy the service"
	synthesis := "Use Docker for deployment"
	responses := map[string]string{"agy": "response A"}

	prompt := BuildExecPrompt(task, &synthesis, responses, "agent")

	// Should contain the task.
	assert.Contains(t, prompt, task)
	// Should contain the synthesis.
	assert.Contains(t, prompt, synthesis)
	// Agent mode template should mention tools/execution.
	assert.Contains(t, prompt, "tools", "agent mode prompt should mention tools")
}

func TestBuildExecPromptNoSynthesisFallsBackToResponses(t *testing.T) {
	task := "do something"
	responses := map[string]string{
		"agy":  "agy guidance text",
		"grok": "grok guidance text",
	}

	// synthesis = nil → executor reads raw responses.
	prompt := BuildExecPrompt(task, nil, responses, "code")

	assert.Contains(t, prompt, task)
	// Should contain the raw response texts (not a synthesis).
	assert.Contains(t, prompt, "agy guidance text")
	assert.Contains(t, prompt, "grok guidance text")
	// Should NOT have a separate synthesis section — the responses ARE the guidance.
	// Model names should NOT appear as labels (anonymized).
	assert.NotContains(t, prompt, "Response from agy",
		"executor prompt should use anonymous labels, not model names")
	assert.NotContains(t, prompt, "Response from grok",
		"executor prompt should use anonymous labels, not model names")
	// Anonymous labels should be present.
	assert.Contains(t, prompt, "Response A")
	assert.Contains(t, prompt, "Response B")
}

func TestBuildExecPromptEmptySynthesisFallsBackToResponses(t *testing.T) {
	task := "do something"
	emptySynth := ""
	responses := map[string]string{
		"agy": "agy guidance text",
	}

	// synthesis = &"" (empty string) → should also fall back to responses.
	prompt := BuildExecPrompt(task, &emptySynth, responses, "code")

	assert.Contains(t, prompt, "agy guidance text",
		"empty synthesis should fall back to raw responses")
}

func TestBuildExecPromptUnknownModeFallsBackToAgent(t *testing.T) {
	task := "task"
	synthesis := "synth text"
	responses := map[string]string{"agy": "resp"}

	// Unknown mode should fall back to the agent template (safest superset).
	prompt := BuildExecPrompt(task, &synthesis, responses, "unknown-mode")

	assert.Contains(t, prompt, task)
	assert.Contains(t, prompt, synthesis)
	// Agent template mentions "tools".
	assert.Contains(t, prompt, "tools")
}

func TestBuildExecPromptCodeVsAgentDifferentTemplates(t *testing.T) {
	task := "task"
	synthesis := "synth"
	responses := map[string]string{"agy": "resp"}

	codePrompt := BuildExecPrompt(task, &synthesis, responses, "code")
	agentPrompt := BuildExecPrompt(task, &synthesis, responses, "agent")

	// The two prompts should differ (different instruction text).
	assert.NotEqual(t, codePrompt, agentPrompt,
		"code and agent modes should use different templates")
	// Code template should contain "Implement the solution".
	assert.Contains(t, codePrompt, "Implement the solution")
	// Agent template should contain "Execute the task now".
	assert.Contains(t, agentPrompt, "Execute the task now")
}

// ---------------------------------------------------------------------------
// anonymousLabel tests
// ---------------------------------------------------------------------------

func TestAnonymousLabel(t *testing.T) {
	assert.Equal(t, "A", anonymousLabel(0))
	assert.Equal(t, "B", anonymousLabel(1))
	assert.Equal(t, "Z", anonymousLabel(25))
	assert.Equal(t, "AA", anonymousLabel(26))
	assert.Equal(t, "AB", anonymousLabel(27))
	assert.Equal(t, "?", anonymousLabel(-1))
}

func TestBuildSynthesisPromptAnonymityInstruction(t *testing.T) {
	responses := map[string]string{
		"agy":  "response A content",
		"grok": "response B content",
	}

	prompt := BuildSynthesisPrompt("task", responses)

	// The template should mention anonymity to guide the synthesizer.
	assert.Contains(t, prompt, "anonymously",
		"synthesis prompt should mention anonymity to prevent bias")
}
