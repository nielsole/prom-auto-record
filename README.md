# prom-auto-record



Automatically creating recording rules

rates are assumed to be dynamically set by Grafana dashboards. Instead, the rate range modifier is always set to the length of the recording rule evaluation interval.
At query time, the original rate is replaced by an avg_over_time(...[$requested_interval])
Sum and avg should be safe operations that can be included in the recording rule.

The Walker walks through the query AST and identifies pieces that are safe to process via a recording rule.

The largest subtree that is recordable is being put into a recording rule.
A future improvement could be to reuse subtrees / recording rules.


