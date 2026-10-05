|                        Metric |                Task |    Baseline |   Contender |        Diff |   Unit |    Diff % |
|------------------------------:|--------------------:|------------:|------------:|------------:|-------:|----------:|
|       Total Young Gen GC time |                     |    0        |    0        |     0       |      s |     0.00% |
|      Total Young Gen GC count |                     |    0        |    0        |     0       |        |     0.00% |
|         Total Old Gen GC time |                     |    0        |    0        |     0       |      s |     0.00% |
|        Total Old Gen GC count |                     |    0        |    0        |     0       |        |     0.00% |
|   Total Ingest Pipeline count |                     |    0        |    0        |     0       |        |     0.00% |
|    Total Ingest Pipeline time |                     |    0        |    0        |     0       |     ms |     0.00% |
|  Total Ingest Pipeline failed |                     |    0        |    0        |     0       |        |     0.00% |
|                Min Throughput |          bulk-index | 2206.1      |  815.992    | -1390.11    | docs/s |   -63.01% |
|               Mean Throughput |          bulk-index | 2347.46     |  876.841    | -1470.62    | docs/s |   -62.65% |
|             Median Throughput |          bulk-index | 2347.46     |  864.023    | -1483.44    | docs/s |   -63.19% |
|                Max Throughput |          bulk-index | 2488.83     |  977.231    | -1511.6     | docs/s |   -60.74% |
|       50th percentile latency |          bulk-index |  118.595    |  261.157    |   142.562   |     ms |  +120.21% |
|       90th percentile latency |          bulk-index |  135.729    |  359.054    |   223.325   |     ms |  +164.54% |
|      100th percentile latency |          bulk-index |  144.82     |  428.545    |   283.725   |     ms |  +195.92% |
|  50th percentile service time |          bulk-index |  118.595    |  261.157    |   142.562   |     ms |  +120.21% |
|  90th percentile service time |          bulk-index |  135.729    |  359.054    |   223.325   |     ms |  +164.54% |
| 100th percentile service time |          bulk-index |  144.82     |  428.545    |   283.725   |     ms |  +195.92% |
|                    error rate |          bulk-index |    0        |    0        |     0       |      % |     0.00% |
|                Min Throughput |     match-all-query |   46.7647   |   34.4809   |   -12.2838  |  ops/s |   -26.27% |
|               Mean Throughput |     match-all-query |   47.2791   |   37.6484   |    -9.6307  |  ops/s |   -20.37% |
|             Median Throughput |     match-all-query |   47.3207   |   37.7937   |    -9.52704 |  ops/s |   -20.13% |
|                Max Throughput |     match-all-query |   48.0695   |   39.9586   |    -8.11085 |  ops/s |   -16.87% |
|       50th percentile latency |     match-all-query |   42.4919   |   47.1557   |     4.66379 |     ms |   +10.98% |
|       90th percentile latency |     match-all-query |   48.3577   |   61.6461   |    13.2884  |     ms |   +27.48% |
|       99th percentile latency |     match-all-query |   59.0105   |  101.797    |    42.7865  |     ms |   +72.51% |
|      100th percentile latency |     match-all-query |  147.689    |  132.963    |   -14.726   |     ms |    -9.97% |
|  50th percentile service time |     match-all-query |   42.4919   |   47.1557   |     4.66379 |     ms |   +10.98% |
|  90th percentile service time |     match-all-query |   48.3577   |   61.6461   |    13.2884  |     ms |   +27.48% |
|  99th percentile service time |     match-all-query |   59.0105   |  101.797    |    42.7865  |     ms |   +72.51% |
| 100th percentile service time |     match-all-query |  147.689    |  132.963    |   -14.726   |     ms |    -9.97% |
|                    error rate |     match-all-query |    0        |    0        |     0       |      % |     0.00% |
|                Min Throughput |   match-title-query | 2894.66     |  453.94     | -2440.72    |  ops/s |   -84.32% |
|               Mean Throughput |   match-title-query | 2894.66     |  453.94     | -2440.72    |  ops/s |   -84.32% |
|             Median Throughput |   match-title-query | 2894.66     |  453.94     | -2440.72    |  ops/s |   -84.32% |
|                Max Throughput |   match-title-query | 2894.66     |  453.94     | -2440.72    |  ops/s |   -84.32% |
|       50th percentile latency |   match-title-query |    0.401563 |    3.05906  |     2.6575  |     ms |  +661.79% |
|       90th percentile latency |   match-title-query |    0.713278 |    5.21362  |     4.50034 |     ms |  +630.94% |
|       99th percentile latency |   match-title-query |    1.67582  |   12.7094   |    11.0335  |     ms |  +658.40% |
|      100th percentile latency |   match-title-query |    2.10312  |   40.8794   |    38.7763  |     ms | +1843.75% |
|  50th percentile service time |   match-title-query |    0.401563 |    3.05906  |     2.6575  |     ms |  +661.79% |
|  90th percentile service time |   match-title-query |    0.713278 |    5.21362  |     4.50034 |     ms |  +630.94% |
|  99th percentile service time |   match-title-query |    1.67582  |   12.7094   |    11.0335  |     ms |  +658.40% |
| 100th percentile service time |   match-title-query |    2.10312  |   40.8794   |    38.7763  |     ms | +1843.75% |
|                    error rate |   match-title-query |    0        |    0        |     0       |      % |     0.00% |
|                Min Throughput |  match-phrase-query |  266.381    |  132.958    |  -133.423   |  ops/s |   -50.09% |
|               Mean Throughput |  match-phrase-query |  279.521    |  169.643    |  -109.878   |  ops/s |   -39.31% |
|             Median Throughput |  match-phrase-query |  279.521    |  177.015    |  -102.506   |  ops/s |   -36.67% |
|                Max Throughput |  match-phrase-query |  292.661    |  198.957    |   -93.7042  |  ops/s |   -32.02% |
|       50th percentile latency |  match-phrase-query |    5.16069  |    7.46719  |     2.3065  |     ms |   +44.69% |
|       90th percentile latency |  match-phrase-query |    9.19599  |   10.6458   |     1.4498  |     ms |   +15.77% |
|       99th percentile latency |  match-phrase-query |   14.4726   |   20.1686   |     5.69595 |     ms |   +39.36% |
|      100th percentile latency |  match-phrase-query |   29.3007   |   23.1275   |    -6.17312 |     ms |   -21.07% |
|  50th percentile service time |  match-phrase-query |    5.16069  |    7.46719  |     2.3065  |     ms |   +44.69% |
|  90th percentile service time |  match-phrase-query |    9.19599  |   10.6458   |     1.4498  |     ms |   +15.77% |
|  99th percentile service time |  match-phrase-query |   14.4726   |   20.1686   |     5.69595 |     ms |   +39.36% |
| 100th percentile service time |  match-phrase-query |   29.3007   |   23.1275   |    -6.17312 |     ms |   -21.07% |
|                    error rate |  match-phrase-query |    0        |    0        |     0       |      % |     0.00% |
|                Min Throughput | term-category-query | 7891.64     | 8829.71     |   938.066   |  ops/s |   +11.89% |
|               Mean Throughput | term-category-query | 7891.64     | 8829.71     |   938.066   |  ops/s |   +11.89% |
|             Median Throughput | term-category-query | 7891.64     | 8829.71     |   938.066   |  ops/s |   +11.89% |
|                Max Throughput | term-category-query | 7891.64     | 8829.71     |   938.066   |  ops/s |   +11.89% |
|       50th percentile latency | term-category-query |    0.129959 |    0.110562 |    -0.0194  |     ms |   -14.93% |
|       90th percentile latency | term-category-query |    0.164729 |    0.146141 |    -0.01859 |     ms |   -11.28% |
|       99th percentile latency | term-category-query |    0.275983 |    0.274798 |    -0.00118 |     ms |    -0.43% |
|      100th percentile latency | term-category-query |    0.479874 |    0.463125 |    -0.01675 |     ms |    -3.49% |
|  50th percentile service time | term-category-query |    0.129959 |    0.110562 |    -0.0194  |     ms |   -14.93% |
|  90th percentile service time | term-category-query |    0.164729 |    0.146141 |    -0.01859 |     ms |   -11.28% |
|  99th percentile service time | term-category-query |    0.275983 |    0.274798 |    -0.00118 |     ms |    -0.43% |
| 100th percentile service time | term-category-query |    0.479874 |    0.463125 |    -0.01675 |     ms |    -3.49% |
|                    error rate | term-category-query |    0        |    0        |     0       |      % |     0.00% |
|                Min Throughput | numeric-range-query |  114.662    |   96.9666   |   -17.6955  |  ops/s |   -15.43% |
|               Mean Throughput | numeric-range-query |  120.27     |  103.327    |   -16.9433  |  ops/s |   -14.09% |
|             Median Throughput | numeric-range-query |  121.777    |  104.114    |   -17.6638  |  ops/s |   -14.50% |
|                Max Throughput | numeric-range-query |  124.371    |  108.114    |   -16.2573  |  ops/s |   -13.07% |
|       50th percentile latency | numeric-range-query |   13.7417   |   16.518    |     2.77631 |     ms |   +20.20% |
|       90th percentile latency | numeric-range-query |   17.4772   |   19.7015   |     2.22428 |     ms |   +12.73% |
|       99th percentile latency | numeric-range-query |   22.6835   |   30.6231   |     7.93952 |     ms |   +35.00% |
|      100th percentile latency | numeric-range-query |   27.5596   |   33.0671   |     5.5075  |     ms |   +19.98% |
|  50th percentile service time | numeric-range-query |   13.7417   |   16.518    |     2.77631 |     ms |   +20.20% |
|  90th percentile service time | numeric-range-query |   17.4772   |   19.7015   |     2.22428 |     ms |   +12.73% |
|  99th percentile service time | numeric-range-query |   22.6835   |   30.6231   |     7.93952 |     ms |   +35.00% |
| 100th percentile service time | numeric-range-query |   27.5596   |   33.0671   |     5.5075  |     ms |   +19.98% |
|                    error rate | numeric-range-query |    0        |    0        |     0       |      % |     0.00% |
|                Min Throughput | bool-compound-query | 7093.3      | 6683.81     |  -409.488   |  ops/s |    -5.77% |
|               Mean Throughput | bool-compound-query | 7093.3      | 6683.81     |  -409.488   |  ops/s |    -5.77% |
|             Median Throughput | bool-compound-query | 7093.3      | 6683.81     |  -409.488   |  ops/s |    -5.77% |
|                Max Throughput | bool-compound-query | 7093.3      | 6683.81     |  -409.488   |  ops/s |    -5.77% |
|       50th percentile latency | bool-compound-query |    0.138124 |    0.139896 |     0.00177 |     ms |    +1.28% |
|       90th percentile latency | bool-compound-query |    0.164746 |    0.175191 |     0.01045 |     ms |    +6.34% |
|       99th percentile latency | bool-compound-query |    0.281491 |    0.33146  |     0.04997 |     ms |   +17.75% |
|      100th percentile latency | bool-compound-query |    0.431832 |    0.522832 |     0.091   |     ms |   +21.07% |
|  50th percentile service time | bool-compound-query |    0.138124 |    0.139896 |     0.00177 |     ms |    +1.28% |
|  90th percentile service time | bool-compound-query |    0.164746 |    0.175191 |     0.01045 |     ms |    +6.34% |
|  99th percentile service time | bool-compound-query |    0.281491 |    0.33146  |     0.04997 |     ms |   +17.75% |
| 100th percentile service time | bool-compound-query |    0.431832 |    0.522832 |     0.091   |     ms |   +21.07% |
|                    error rate | bool-compound-query |    0        |    0        |     0       |      % |     0.00% |
