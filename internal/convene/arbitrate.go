// arbitrate.go — Multi-model voting panel (omnilane-inspired arbitrate lane).
//
// When synthesis_mode == "vote", the engine replaces the single-synthesizer
// reasoning step with an arbitrate panel:
//
//	Round 1: the task goes to every voter (1-4 models). Each answers
//	  independently (read-only Respond).
//	Round 2 (optional, when vote_rounds >= 2): every voter sees all
//	  round-1 opinions and rebuts only the disagreements.
//
// A chair model (the synthesizer, or the first voter if no synthesizer is
// configured) then assembles the final verdict from the panel output.
//
// The panel is fault-tolerant: a single voter failure does not abort the
// panel — the chair works with whatever opinions came back. If all voters
// fail, ArbitratePanel returns an error and the engine falls back to the
// raw responder responses (same nil-synthesis semantics as a failed
// synthesizer).
package convene

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
)

// votePromptTemplate is sent to each voter in round 1.
const votePromptTemplate = `You are a voter on an arbitrate panel in a Mixture-of-Agents pipeline. Multiple AI models have independently responded to the same task. Your job is to give your own independent verdict, taking their responses into account.

INSTRUCTIONS:
- Read the task and the model responses below.
- Form your own opinion. Do not simply average or echo the others.
- If you disagree with a response, say so and explain why.
- Be concise but complete.

=== ORIGINAL TASK ===
%s

=== MODEL RESPONSES ===
%s

=== YOUR VERDICT ===`

// rebuttalPromptTemplate is sent to each voter in round 2 (debate round).
const rebuttalPromptTemplate = `You are a voter on an arbitrate panel in a Mixture-of-Agents pipeline. This is the debate round. Below are the independent verdicts from all panel members (including your own round-1 verdict). Your job is to rebut only the points you disagree with and reaffirm or revise your position.

INSTRUCTIONS:
- Focus on disagreements. Do not restate points everyone already agrees on.
- If another voter changed your mind, say so explicitly.
- End with your final position.

=== ORIGINAL TASK ===
%s

=== PANEL VERDICTS (round 1) ===
%s

=== YOUR REBUTTAL + FINAL POSITION ===`

// chairPromptTemplate is sent to the chair model to assemble the final verdict.
const chairPromptTemplate = `You are the chair of an arbitrate panel in a Mixture-of-Agents pipeline. The panel has debated the task below. Your job is to assemble the final verdict.

INSTRUCTIONS:
- Read all panel verdicts (and rebuttals, if a debate round happened).
- Identify where the panel agreed and where it disagreed.
- Commit to the best-supported position on each disputed point.
- Do NOT average or majority-vote — reason to the strongest conclusion.
- Produce the final answer.

=== ORIGINAL TASK ===
%s

=== PANEL OUTPUT ===
%s

=== FINAL VERDICT ===`

// ArbitratePanel runs a multi-model voting panel and returns the assembled
// verdict.
//
// Parameters:
//   - ctx: context for cancellation.
//   - task: the original task.
//   - responses: the round-1 responder outputs (anonymous, labeled A/B/C...).
//   - voters: voter model names (1-4). If empty, the responder names are used.
//   - chair: the chair model name (assembles the final verdict). If empty,
//     the first voter is used.
//   - rounds: 1 = single round, 2 = debate round.
//   - timeout: per-call timeout (0 = use config default).
//
// Returns the assembled verdict string and metadata. On total voter failure
// returns an error (caller falls back to nil synthesis).
func (e *ConveneEngine) ArbitratePanel(
	ctx context.Context,
	task string,
	responses map[string]string,
	voters []string,
	chair string,
	rounds int,
	timeout int,
) (string, map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	start := time.Now()

	if len(voters) == 0 {
		// Reuse responder names as voters.
		for name := range responses {
			voters = append(voters, name)
		}
		sort.Strings(voters)
	}
	if len(voters) == 0 {
		return "", metadata, fmt.Errorf("arbitrate panel: no voters available")
	}
	if chair == "" {
		chair = voters[0]
	}
	if rounds < 1 {
		rounds = 1
	}

	responsesSection := buildResponsesSection(responses)

	// --- Round 1: each voter answers independently. ---
	round1 := e.fanOutVoters(ctx, task, votePromptTemplate, task, responsesSection, voters, timeout, metadata, "round1")
	if len(round1) == 0 {
		metadata["arbitrate_elapsed"] = time.Since(start)
		return "", metadata, fmt.Errorf("arbitrate panel: all round-1 voters failed")
	}

	panelOutput := round1

	// --- Round 2 (optional): debate round. ---
	if rounds >= 2 && len(round1) > 1 {
		verdictsSection := buildVerdictsSection(round1)
		round2 := e.fanOutVoters(ctx, task, rebuttalPromptTemplate, task, verdictsSection, voters, timeout, metadata, "round2")
		if len(round2) > 0 {
			panelOutput = round2
		}
	}

	// --- Chair assembles the final verdict. ---
	verdictsSection := buildVerdictsSection(panelOutput)
	chairPrompt := fmt.Sprintf(chairPromptTemplate, task, verdictsSection)

	chairCfg, exists := e.Config.Models[chair]
	if !exists {
		dynCfg, _, dynOk := adapter.ResolveDynamicModel(chair)
		if !dynOk {
			metadata["arbitrate_elapsed"] = time.Since(start)
			metadata["arbitrate_error"] = fmt.Sprintf("chair %s not found", chair)
			// Fallback: return the first verdict as the synthesis.
			for _, v := range panelOutput {
				metadata["arbitrate_elapsed"] = time.Since(start)
				return v, metadata, nil
			}
		} else {
			chairCfg = dynCfg
		}
	}

	chairAdapter, err := e.adapterFactory(chair, chairCfg)
	if err != nil {
		metadata["arbitrate_elapsed"] = time.Since(start)
		metadata["arbitrate_error"] = fmt.Sprintf("chair adapter failed: %v", err)
		// Fallback: return the first verdict.
		for _, v := range panelOutput {
			metadata["arbitrate_elapsed"] = time.Since(start)
			return v, metadata, nil
		}
		return "", metadata, fmt.Errorf("arbitrate panel: chair adapter failed and no verdicts to fall back on")
	}

	t := timeout
	if t <= 0 {
		t = chairCfg.Timeout
	}
	if t <= 0 {
		t = e.Config.Defaults.Timeout
	}

	chairStart := time.Now()
	chairResult, err := chairAdapter.Respond(ctx, chairPrompt, t)
	metadata["arbitrate_chair_elapsed"] = time.Since(chairStart)

	if err != nil || !chairResult.Success {
		metadata["arbitrate_elapsed"] = time.Since(start)
		metadata["arbitrate_error"] = fmt.Sprintf("chair call failed: %v", err)
		// Fallback: return the first verdict.
		for _, v := range panelOutput {
			metadata["arbitrate_elapsed"] = time.Since(start)
			return v, metadata, nil
		}
		return "", metadata, fmt.Errorf("arbitrate panel: chair failed and no verdicts to fall back on")
	}

	metadata["arbitrate_elapsed"] = time.Since(start)
	metadata["arbitrate_voters"] = voters
	metadata["arbitrate_rounds"] = rounds
	metadata["arbitrate_chair"] = chair
	metadata["arbitrate_success"] = true
	return chairResult.Stdout, metadata, nil
}

// fanOutVoters runs a set of voter Respond calls in parallel and collects
// the successful verdicts. Voter names are anonymized in the output map
// (Voter A, Voter B, ...) to prevent bias.
func (e *ConveneEngine) fanOutVoters(
	ctx context.Context,
	task string,
	template string,
	taskArg string,
	guidanceArg string,
	voters []string,
	timeout int,
	metadata map[string]interface{},
	roundTag string,
) map[string]string {
	type voterResult struct {
		name    string
		stdout  string
		ok      bool
		elapsed time.Duration
	}

	results := make([]voterResult, len(voters))
	var wg sync.WaitGroup
	for i, name := range voters {
		i, name := i, name
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()

			cfg, exists := e.Config.Models[name]
			if !exists {
				dynCfg, _, dynOk := adapter.ResolveDynamicModel(name)
				if !dynOk {
					results[i] = voterResult{name: name, ok: false}
					return
				}
				cfg = dynCfg
			}

			a, err := e.adapterFactory(name, cfg)
			if err != nil {
				results[i] = voterResult{name: name, ok: false, elapsed: time.Since(start)}
				return
			}

			t := timeout
			if t <= 0 {
				t = cfg.Timeout
			}
			if t <= 0 {
				t = e.Config.Defaults.Timeout
			}

			prompt := fmt.Sprintf(template, taskArg, guidanceArg)
			r, err := a.Respond(ctx, prompt, t)
			results[i] = voterResult{
				name:    name,
				stdout:  r.Stdout,
				ok:      err == nil && r.Success,
				elapsed: time.Since(start),
			}
		}()
	}
	wg.Wait()

	// Collect successful verdicts, anonymized.
	out := make(map[string]string)
	for i, r := range results {
		metadata[fmt.Sprintf("arbitrate_%s_%s_elapsed", roundTag, r.name)] = r.elapsed
		metadata[fmt.Sprintf("arbitrate_%s_%s_success", roundTag, r.name)] = r.ok
		if r.ok {
			out[fmt.Sprintf("Voter %s", anonymousLabel(i))] = r.stdout
		}
	}
	return out
}

// buildVerdictsSection assembles voter verdicts into a labeled text block.
func buildVerdictsSection(verdicts map[string]string) string {
	if len(verdicts) == 0 {
		return "(No panel verdicts available.)"
	}
	names := make([]string, 0, len(verdicts))
	for n := range verdicts {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "--- %s ---\n%s", name, verdicts[name])
	}
	return b.String()
}
