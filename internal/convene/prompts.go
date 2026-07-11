// prompts.go — Prompt template construction for the Convene engine.
//
// This file contains:
//   - synthesisPromptTemplate: the synthesizer's instruction template.
//   - execPromptTemplateCode:  the executor's instruction template (code mode).
//   - execPromptTemplateAgent: the executor's instruction template (agent mode).
//   - BuildSynthesisPrompt:    assembles the synthesizer prompt from task + N responses.
//   - BuildExecPrompt:         assembles the executor prompt from task + synthesis (or responses).
//
// MoA design principle (arXiv:2406.04692):
// The synthesis prompt MUST instruct the synthesizer to perform reasoning-based
// integration — NOT majority voting, NOT averaging. The synthesizer reads all
// responder outputs, identifies which model is correct on which sub-argument,
// which has hallucinations, which argument is most complete, and assembles a
// final answer stronger than any single response. Randomness (stochasticity)
// is a feature: parallel multi-model sampling draws multiple samples from the
// response distribution, and the synthesizer picks the optimal combination.
package convene

import (
	"fmt"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Template constants
// ---------------------------------------------------------------------------

// synthesisPromptTemplate is the instruction given to the synthesizer model.
//
// Key MoA directives embedded in this template:
//   - Do NOT average opinions.
//   - Do NOT majority-vote.
//   - DO perform reasoning-based integration.
//   - Identify which model is correct on each sub-argument.
//   - Flag hallucinations and factual errors in individual responses.
//   - Assemble a final answer that is stronger than any single response.
//
// Anonymity: responder outputs are labeled with neutral identifiers
// ("Response A", "Response B", ...) instead of model names. This prevents
// bias when the synthesizer is the same model as one of the responders.
const synthesisPromptTemplate = `You are a synthesis agent in a Mixture-of-Agents (MoA) pipeline. Multiple AI models have independently responded to the same task. Your job is to produce a single, superior answer by reasoning over their responses.

CRITICAL INSTRUCTIONS:
- Do NOT average their opinions.
- Do NOT majority-vote or pick the most common answer.
- DO perform reasoning-based integration: read every response, identify which response is correct on each sub-argument, flag any hallucinations or factual errors, note which response has the most complete reasoning, and assemble a final answer that is stronger than any individual response.
- If one response is clearly correct on a sub-point while others are wrong, adopt the correct one — do not compromise.
- If responses disagree, explain the disagreement briefly, then commit to the best-supported position.
- Stochasticity is expected: different models may produce different styles or levels of detail. Use this to your advantage — combine the strongest parts.
- The responses are labeled anonymously (A, B, C, ...) to prevent bias. Evaluate each response purely on its merits, not on which model produced it.

=== ORIGINAL TASK ===
%s

=== MODEL RESPONSES ===
%s

=== YOUR SYNTHESIS ===
Produce a comprehensive, well-structured answer that integrates the best reasoning from all responses above. Do not reference the response labels in your output — just produce the best possible answer.`

// execPromptTemplateCode is the instruction given to the executor in code mode.
//
// The executor is the ONLY role with side-effects (file writes, command
// execution). It runs AFTER synthesis, as a single execution unit, to prevent
// N models from simultaneously modifying files during fan-out.
const execPromptTemplateCode = `You are the executor agent in a Mixture-of-Agents (MoA) pipeline. A team of models has analyzed the task below, and a synthesis of their responses is provided. Your job is to implement the solution in code.

INSTRUCTIONS:
- Use the synthesis (or the model responses, if no synthesis is available) as your primary guidance.
- Write clean, working code that fulfills the task requirements.
- If the synthesis contains specific implementation details, follow them unless they are clearly wrong.
- You are the only agent with write access — execute confidently based on the integrated guidance.
- When raw responses are provided (no synthesis), they are labeled anonymously (A, B, C, ...) to prevent bias. Evaluate each on its merits.

=== ORIGINAL TASK ===
%s

=== INTEGRATED GUIDANCE ===
%s

=== YOUR IMPLEMENTATION ===
Implement the solution now. Write code, create or modify files as needed, and run any necessary commands to verify your work.`

// execPromptTemplateAgent is the instruction given to the executor in agent mode.
//
// Agent mode is broader than code mode: the executor may perform any agentic
// action (research, file operations, command execution, multi-step workflows).
const execPromptTemplateAgent = `You are the executor agent in a Mixture-of-Agents (MoA) pipeline. A team of models has analyzed the task below, and a synthesis of their responses is provided. Your job is to execute the task using all available tools.

INSTRUCTIONS:
- Use the synthesis (or the model responses, if no synthesis is available) as your primary guidance.
- Perform all necessary actions to complete the task: research, file operations, command execution, multi-step workflows.
- If the synthesis contains a specific action plan, follow it unless it is clearly wrong.
- You are the only agent with write access — execute confidently based on the integrated guidance.
- When raw responses are provided (no synthesis), they are labeled anonymously (A, B, C, ...) to prevent bias. Evaluate each on its merits.

=== ORIGINAL TASK ===
%s

=== INTEGRATED GUIDANCE ===
%s

=== YOUR EXECUTION ===
Execute the task now. Use all available tools as needed and report what you did.`

// ---------------------------------------------------------------------------
// Prompt builder functions
// ---------------------------------------------------------------------------

// BuildSynthesisPrompt assembles the synthesizer's prompt from the original
// task and the N responder responses.
//
// The prompt includes:
//   1. The MoA integration instructions (synthesisPromptTemplate).
//   2. The original task.
//   3. Each responder's response, labeled with a neutral anonymous identifier
//      ("Response A", "Response B", ...) — model names are NOT included.
//      This prevents bias when the synthesizer is the same model as one of
//      the responders.
//
// Responses are sorted by model name (internally) for deterministic ordering,
// but only the anonymous label (A, B, C, ...) is shown to the synthesizer.
//
// The template explicitly instructs the synthesizer to perform reasoning-based
// integration, NOT majority voting or averaging — this is the core MoA
// principle that makes synthesis stronger than any single model.
func BuildSynthesisPrompt(task string, responses map[string]string) string {
	// Sort model names for deterministic prompt ordering (stable output).
	names := make([]string, 0, len(responses))
	for name := range responses {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteString("\n\n")
		}
		// Anonymous label: A, B, C, ... Z, AA, AB, ...
		label := anonymousLabel(i)
		fmt.Fprintf(&b, "--- Response %s ---\n%s", label, responses[name])
	}

	return fmt.Sprintf(synthesisPromptTemplate, task, b.String())
}

// BuildExecPrompt assembles the executor's prompt from the original task and
// either the synthesis (if available) or the raw responder responses.
//
// Behavior:
//   - If synthesis != nil → the integrated guidance section uses the synthesis
//     text. This is the preferred path: the executor acts on the synthesizer's
//     reasoning-based integration.
//   - If synthesis == nil → the integrated guidance section assembles the raw
//     responder responses (labeled by model name, sorted for stability). This
//     happens when no synthesizer was configured (executor doubles as
//     synthesizer) or the synthesizer failed.
//
// The mode parameter ("code" | "agent") selects the instruction template:
//   - "code"  → execPromptTemplateCode (implementation-focused).
//   - "agent" → execPromptTemplateAgent (broader agentic actions).
//   - other   → falls back to the agent template (safest superset).
func BuildExecPrompt(task string, synthesis *string, responses map[string]string, mode string) string {
	// Determine the integrated guidance content.
	var guidance string
	if synthesis != nil && *synthesis != "" {
		guidance = *synthesis
	} else {
		// No synthesis available — assemble raw responses as guidance.
		guidance = buildResponsesSection(responses)
	}

	// Select the template based on mode.
	var tmpl string
	switch mode {
	case "code":
		tmpl = execPromptTemplateCode
	default:
		// "agent" and any unknown mode use the agent template (safest superset).
		tmpl = execPromptTemplateAgent
	}

	return fmt.Sprintf(tmpl, task, guidance)
}

// buildResponsesSection assembles the raw responder responses into a single
// text block, labeled with anonymous identifiers (A, B, C, ...) and sorted
// by model name (internally) for deterministic ordering.
//
// Model names are NOT shown to the executor — only anonymous labels. This
// prevents bias when the executor is the same model as one of the responders.
//
// This is used by BuildExecPrompt when no synthesis is available (executor
// doubles as synthesizer, or synthesizer failed).
func buildResponsesSection(responses map[string]string) string {
	if len(responses) == 0 {
		return "(No responder responses available.)"
	}

	// Sort model names for deterministic ordering.
	names := make([]string, 0, len(responses))
	for name := range responses {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("The following are independent responses from multiple models. Use them as guidance:\n")
	for i, name := range names {
		if i > 0 {
			b.WriteString("\n\n")
		}
		label := anonymousLabel(i)
		fmt.Fprintf(&b, "--- Response %s ---\n%s", label, responses[name])
	}
	return b.String()
}

// anonymousLabel converts a 0-based index to an anonymous label: 0→A, 1→B,
// 25→Z, 26→AA, 27→AB, ... This provides neutral identifiers that do not
// reveal the underlying model name, preventing synthesizer/executor bias
// when they share the same model as a responder.
func anonymousLabel(index int) string {
	if index < 0 {
		return "?"
	}
	var b []byte
	for {
		b = append([]byte{byte('A' + index%26)}, b...)
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	return string(b)
}
