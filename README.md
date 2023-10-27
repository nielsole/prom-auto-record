# Automatic Prometheus Recording rules

## Overview

`prom-auto-record` is an automation tool for generating Prometheus recording rules from a set of existing Prometheus queries.
Manual creation of recording rules for Grafana dashboards can be tedious and laborous.
This project sets out to solve that.

The idea would be to add an HTTP server to this program that would act as a proxy between grafana and prometheus.
After assembling information on the queries it would then write recording rules to a configmap which would in turn be read by prometheus itself.
At query-time this proxy would then analyze queries and replace them with recordings. If a recording rule was recently added and not enough data has been recorded, the query would get passed through to upstream.

**This project never left the POC stage**

## Goal

The primary goal is to automatically identify portions of a given Prometheus query that can be pre-computed and stored as a recording rule. This is particularly useful for complex and expensive Grafana queries, which often involve complicated selectors and aggregate functions.

## Strategy

1. **Identifying Safe Subtrees**: The first step is to walk through the Prometheus query AST (Abstract Syntax Tree) to identify "safe" subtrees. A safe subtree is a portion of the query that can safely be replaced by a recording rule without changing the query's meaning or results.

2. **Signature Generation**: Once safe subtrees are identified, we generate a "signature" for each subtree. This signature helps in recognizing similar query parts across different queries, enabling us to reuse recording rules effectively.

3. **Metric Naming**: A unique, collision-free metric name is generated for each recording rule. This is done by taking a hash of the generated signature and appending it to a static prefix.

4. **Creating Recording Rules**: Finally, the recording rules are created based on these safe subtrees. Currently, these recording rules are minimal, aiming for a "Minimum Viable Product" that serves as a proof of concept.

### Rate Ranges

Rate ranges are usually set dynamically by Grafana dashboards. However, in our approach, the rate range modifier for the recording rule is set to the length of the recording rule's evaluation interval. At query time, the original rate is replaced by an `avg_over_time(...[$requested_interval])`.

### Safe Operations

Currently, `VectorSelector`, `sum`, `count` and `avg` are considered safe operations. Other functions like `topk`, and so forth are not yet supported but are planned for future releases.

## What's Not Implemented Yet

1. **Cardinality, Commonality, and Complexity Estimation**: These metrics for deciding whether a subtree should be converted into a recording rule are not yet implemented.
  
2. **Dynamic Rate Ranges**: Customization of rate ranges based on Grafana's dynamic setting is not yet supported.

3. **Advanced Aggregate Functions**: Support for more advanced aggregate functions and query features is not yet implemented.

4. **Subtree Reusability**: Currently, the largest "safe" subtree is used for generating a recording rule. The ability to reuse smaller subtrees in multiple recording rules is not yet available.

5. **HTTP Proxy server**

6. **Writing to configmap**

## Usage

Run the program and input your Prometheus queries line-by-line to the standard input. The program will output identified safe subtrees and their corresponding recording rule signatures.

```sh
echo 'topk(5, sum(http_request_duration_seconds_bucket{service="service-b"}) by (le))' | go run .
```

You'll receive an output like this:

```
Enter queries, one per line (Ctrl-D to terminate):
Expr: sum by (le) (http_request_duration_seconds_bucket{service="service-b"})
Signature: sum_by(le)__http_request_duration_seconds_bucket{service=service-b,__name__=http_request_duration_seconds_bucket}_
HashedMetricName: recording_rule_3d52752d9da0
```

## Contributing

This is a POC and while I like hearing from you if you found this interesting, it will likely not see further attention from me. Invest your time at your own peril.

