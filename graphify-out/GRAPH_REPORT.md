# Graph Report - altfins-cli  (2026-06-07)

## Corpus Check
- 38 files · ~25,459 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 559 nodes · 1143 edges · 29 communities (28 shown, 1 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 106 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f662ca73`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]

## God Nodes (most connected - your core abstractions)
1. `browserModel` - 45 edges
2. `Client` - 25 edges
3. `NewRootCommand()` - 19 edges
4. `Enum Appendix` - 19 edges
5. `newTestBrowserModel()` - 18 edges
6. `Context` - 17 edges
7. `paths` - 16 edges
8. `Cmd` - 15 edges
9. `T` - 14 edges
10. `testPageMsg()` - 14 edges

## Surprising Connections (you probably didn't know these)
- `Execute()` --calls--> `FormatError()`  [INFERRED]
  cmd/root.go → internal/app/runtime.go
- `newAuthCommand()` --calls--> `MaskSecret()`  [INFERRED]
  cmd/auth.go → internal/app/runtime.go
- `loadBodyFlags()` --calls--> `LoadJSONObject()`  [INFERRED]
  cmd/helpers.go → internal/app/filters.go
- `factoryFor()` --calls--> `FactoryFromContext()`  [INFERRED]
  cmd/helpers.go → internal/app/context.go
- `main()` --calls--> `Execute()`  [INFERRED]
  main.go → cmd/root.go

## Import Cycles
- None detected.

## Communities (29 total, 1 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (74): content, description, content, description, content, description, content, description (+66 more)

### Community 1 - "Community 1"
Cohesion: 0.08
Nodes (24): chartConfig, chartMode, chartPreset, chartState, Cmd, Color, Client, PermitsInfo (+16 more)

### Community 2 - "Community 2"
Cohesion: 0.10
Nodes (23): APIError, Client, NewClient(), ClientConfig, DryRunError, AnalyticsHistoryData, AnalyticsType, Paging (+15 more)

### Community 3 - "Community 3"
Cohesion: 0.17
Nodes (28): cloneMap(), flattenJSONObject(), mapAssets(), normalizeJSONValue(), ohlcvRows(), projectJSONAny(), projectJSONArray(), projectJSONItems() (+20 more)

### Community 4 - "Community 4"
Cohesion: 0.16
Nodes (27): Duration, OHLCVData, Time, OHLCVData, T, LabelFormatter, chartTimeLabelFormatter(), fallback() (+19 more)

### Community 5 - "Community 5"
Cohesion: 0.09
Nodes (17): IsDryRun(), TestDryRunReturnsPreview(), TestSignalsSearchBuildsRequest(), roundTripFunc, AuthRequiredError, ConfirmationRequiredError, Factory, RootOptions (+9 more)

### Community 6 - "Community 6"
Cohesion: 0.24
Nodes (27): browserModel, browserItem, browserPage, Paging, T, pageMsg, newTestBrowserModel(), TestAutoLoadNearEndLoadsNextPageOnce() (+19 more)

### Community 7 - "Community 7"
Cohesion: 0.11
Nodes (20): NormalizeTimeInput(), ParseCSV(), Command, newAnalyticsCommand(), clientFor(), csvValues(), Client, handleResult() (+12 more)

### Community 8 - "Community 8"
Cohesion: 0.16
Nodes (12): TestFactoryNewClientAllowsDryRunWithoutAPIKey(), TestFactoryNewClientRequiresAPIKeyOutsideDryRun(), DefaultPath(), NewManager(), NewManagerAt(), TestResolveUsesEnvOverConfig(), TestSaveAPIKeyUsesStrictPermissions(), Manager (+4 more)

### Community 9 - "Community 9"
Cohesion: 0.22
Nodes (21): browserItem, browserPage, Client, Context, Dependencies, Page, Paging, Runner (+13 more)

### Community 10 - "Community 10"
Cohesion: 0.13
Nodes (17): AnalyticsHistoryData, AnalyticsType, AssetInfo, NewsSummary, OHLCVData, OrderModel, Page, PermitsInfo (+9 more)

### Community 11 - "Community 11"
Cohesion: 0.15
Nodes (15): factoryFor(), Command, Factory, Paging, Writer, loadBodyFlags(), mustJSON(), writePlainTable() (+7 more)

### Community 12 - "Community 12"
Cohesion: 0.11
Nodes (19): 52-Week Filter, ATH Distance Filter, Candlestick Lookback Intervals, Coin Type Filter, Cross and Comparison Value, Enum Appendix, Intervals, MACD Filter (+11 more)

### Community 13 - "Community 13"
Cohesion: 0.27
Nodes (14): MaskSecret(), Command, newAuthCommand(), annotateEndpoint(), OperationType, annotateSafety(), Command, isInteractiveStdin() (+6 more)

### Community 14 - "Community 14"
Cohesion: 0.34
Nodes (15): cmdNode, authHasKey(), Command, safetyMeta, T, isolatedConfig(), runCLI(), runnableLeaves() (+7 more)

### Community 15 - "Community 15"
Cohesion: 0.20
Nodes (9): exitCoder, Command, newRefsCommand(), Execute(), exitCode(), Command, NewRootCommand(), rootOptions (+1 more)

### Community 16 - "Community 16"
Cohesion: 0.35
Nodes (9): commandMeta, buildCommandMeta(), collectFlagMeta(), flattenCommandMeta(), Command, safetyMeta, newCommandsCommand(), flagMeta (+1 more)

### Community 17 - "Community 17"
Cohesion: 0.20
Nodes (9): 60-Second Quickstart, AI and LLM Friendly by Design, altFINS CLI, Contributing, Features at a Glance, Project Links, Search Guide, Start TUI With Filters (+1 more)

### Community 18 - "Community 18"
Cohesion: 0.22
Nodes (8): Current Channels, Deferred Channels, GitHub Releases, Homebrew, Prepared But Not Yet Public, Public Install Contract, Releasing and Distribution, Winget

### Community 19 - "Community 19"
Cohesion: 0.25
Nodes (7): Artifact Mapping, First-Time Setup, Intended Install Command, Package Identity, Release Flow, Verification, Winget Packaging

### Community 20 - "Community 20"
Cohesion: 0.48
Nodes (6): normalizeNewlines(), TestWriteOutputGolden(), TestWriteOutputJSONFieldsProjectsPageContent(), TestWriteOutputJSONFieldsProjectsPermitsInfo(), TestWriteOutputJSONLFieldsProjectsItems(), T

### Community 21 - "Community 21"
Cohesion: 0.47
Nodes (5): FactoryFromContext(), WithFactory(), contextKey, Context, Factory

### Community 22 - "Community 22"
Cohesion: 0.33
Nodes (5): Command Safety Metadata, Contributing, Local Development, Project Layout, Release Notes

### Community 23 - "Community 23"
Cohesion: 0.33
Nodes (6): Agent-Friendly Dry Run, Auth and First Check, Real CLI Snippets, Signal Hunting, Technical Analysis and News, Typical Table Workflow

### Community 24 - "Community 24"
Cohesion: 0.50
Nodes (3): LoadJSONObject(), parseFilterSource(), Reader

### Community 25 - "Community 25"
Cohesion: 0.40
Nodes (5): Agent Safety Contract, Command safety metadata, Confirmation and force, Dry-run for local writes, Exit codes

### Community 26 - "Community 26"
Cohesion: 0.40
Nodes (5): Common Workflows, Explore reference data, Pull historical analytics, Retrieve OHLCV data, Screen the market

### Community 27 - "Community 27"
Cohesion: 0.40
Nodes (5): Install, Linux, macOS, Optional: WSL + Homebrew, Windows

## Knowledge Gaps
- **136 isolated node(s):** `Command`, `Command`, `safetyMeta`, `Paging`, `Factory` (+131 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Client` connect `Community 2` to `Community 5`?**
  _High betweenness centrality (0.206) - this node is a cross-community bridge._
- **Why does `IsDryRun()` connect `Community 5` to `Community 2`?**
  _High betweenness centrality (0.205) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `NewRootCommand()` (e.g. with `WithFactory()` and `NewFactory()`) actually correct?**
  _`NewRootCommand()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Command`, `Command`, `safetyMeta` to the rest of the system?**
  _136 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.062342342342342344 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.0798611111111111 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.10101010101010101 - nodes in this community are weakly interconnected._