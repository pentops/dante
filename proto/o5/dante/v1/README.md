# Dante

## Dead Message

### State

```mermaid
stateDiagram-v2
    [*] --> CREATED
    CREATED --> UPDATED : edit
    CREATED --> REPLAYED : replay
    CREATED --> REJECTED : reject

    UPDATED --> UPDATED : edit
    UPDATED --> REPLAYED : replay
    UPDATED --> REJECTED : reject

    REJECTED --> [*]
    REPLAYED --> [*]
```
