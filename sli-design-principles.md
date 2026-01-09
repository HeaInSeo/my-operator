### SLI Instrumentation Policy (Design Principles)

This project includes an experimental SLI instrumentation framework intended only for development-time design validation, not for production monitoring.

The following rules are intentional design decisions, not limitations.

1. No operator code modification

This project does not modify operator (controller) source code to insert SLI instrumentation.

- The controller’s Reconcile logic is never touched.  
- No additional metrics are registered inside the controller process.  
- No flags or build-time options are required in the operator code.  

This allows the same SLI instrumentation to be applied to any operator without forking or patching its implementation.

2. **/metrics** is an input source, not an output target

The controller’s /metrics endpoint is used only as a source of raw signals, such as:

- controller_runtime_reconcile_total  
- controller_runtime_reconcile_time_seconds  
- other controller-runtime–provided metrics  

Derived SLI metrics such as:  
- e2e_convergence_time_seconds  
- reconcile_total_delta  

are **not exposed** on the controller’s **/metrics** endpoint.  

This is intentional.  

Because the operator code is not modified, derived SLI metrics cannot and should not be registered in the controller’s Prometheus registry.

3. SLI computation happens in E2E tests

All SLI measurements are computed externally, in E2E test code:

- Start time is recorded when the test creates a primary Custom Resource.  
- End time is recorded when the resource is observed as Ready.  
- Reconcile churn is calculated by reading controller metrics at test start and end.  

This keeps instrumentation:

- independent of operator implementation details  
- resilient to refactoring  
- easy to remove without leaving technical debt
- Default OFF, explicit opt-in  

SLI instrumentation is disabled by default.  

It is enabled only when explicitly requested (for example via environment variables in E2E runs).

If instrumentation is disabled:  

- no measurements are recorded  
- no metrics are scraped  
- test behavior is unchanged  

5. Measurement failure does not fail tests

SLI instrumentation is strictly observational.

Any failure during measurement, including:

- **/metrics** scrape failure  
- metric not found  
- parsing errors  
- timeouts  
- write failures for summary artifacts  

does **not** cause test failure.

Instead:

- the SLI result is marked as skip  
- the test result remains authoritative  

This ensures that experimental instrumentation never destabilizes CI or development workflows.

6. Experiment artifacts

Each E2E run produces, at most, two artifacts:

- **/metrics** (raw controller metrics, scrapeable during the run)  
- sli-summary.json (a per-run summary of derived SLI values)  

These artifacts exist to support design analysis and comparison between runs.
They are not part of a production SLO pipeline.

Summary

This SLI framework is designed to answer one question during development:

“How quietly and predictably does this operator converge?”

It deliberately trades real-time observability for:

- zero operator intrusion  
- maximum reuse across operators  
- minimal long-term maintenance cost  