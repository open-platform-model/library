## MODIFIED Requirements

### Requirement: Each runtime contract has one home

Each runtime contract of the library SHALL be stated in the doc comment of the package, type or function that owns it; the rationale behind a contract SHALL live in an ADR under `adr/`, and its testable obligations in a requirement under `openspec/specs/`. `README.md`, `AGENTS.md` and `docs/getting-started.md` SHALL link to a contract's home instead of restating it, and MAY keep a short orientation sentence that names the home. A doc comment that the docs bundle publishes (any exported package under `opm/`) SHALL NOT carry an `ADR-NNN` pointer; a package that wants one keeps it in a non-doc comment. No committed file outside `openspec/changes/` SHALL cite a session-local decision, one recorded only in an agent run's own log (a numbered decision kept in that log, or an agent role named as the source), as a source: it states the rule and cites a source a reader can open (an ADR; a decision ADR-013 records, cited as `ADR-013, decision c4`; an enhancement decision such as `0012:D3`; a pull request; or an archived change). Source: ADR-013, decision c4.

#### Scenario: The render contract is stated once

- **WHEN** a developer looks for the render gate's cause order outside `opm/`
- **THEN** `README.md`, `AGENTS.md` and `docs/getting-started.md` link to the `opm/kernel` package doc or the `RenderError` doc for it, and none lists the order itself

#### Scenario: The env-override rule is stated in its homes

- **WHEN** a developer searches the non-test Go code under `opm/` for `os.Setenv`
- **THEN** the hits are the `opm/internal/cueenv` package doc and the `OCILoader` type doc; the acquire verbs link `[WithRegistry]` or the `opm/kernel` package doc, and the loader options link `[cueenv.Override]`, instead

#### Scenario: Published doc comments carry no ADR pointer

- **WHEN** a developer searches the doc comments of the exported packages under `opm/` for `ADR-`
- **THEN** none is found; ADR pointers appear only in non-doc comments

#### Scenario: No session-local citation

- **WHEN** a developer runs `git grep -nE '\bSD[0-9]+\b|[Ss]upervisor' -- . ':!openspec/changes'`
- **THEN** it prints nothing; each rule cites a source a reader can open

#### Scenario: A walkthrough decision resolves through ADR-013

- **WHEN** a committed file outside `openspec/changes/` cites a decision of the 2026-10-02/03 kernel plan review, such as the consumer-build rule
- **THEN** it cites it as `ADR-013, decision <id>`, and `adr/013-kernel-plan-walkthrough-decisions.md` holds a row for that id with the decision and the pull requests where it landed

#### Scenario: No walkthrough id without ADR-013

- **WHEN** a developer runs `git grep -nE "[Oo]wner('s)? (walkthrough )?decision [a-j][1-5]\b|walkthrough (decision|task) [a-j][1-5]\b|(beta\.1|kernel[ -]plan) walkthrough" -- . ':!openspec/changes' ':!adr/013-*' ':!CHANGELOG.md'`
- **THEN** it prints nothing
