---
title: "Real-Time Market Data Streaming for Beginners"
date: 2026-09-24T00:00:00-03:00
draft: true
description: "Learn the market-data concepts behind a real-time system for quotes, trades, and price charts."
tags: ["distributed-systems", "market-data", "fintech"]
categories: ["engineering"]
---

Before designing a real-time market-data system, it helps to understand the information it handles. Financial markets are places where people and institutions buy and sell assets: stocks such as Apple shares, options contracts, currencies, cryptocurrencies, and more.

A market-data system does not execute those trades. Its job is to observe what is happening in the market and quickly send that information to applications, traders, charts, and algorithms.

## The small events that move a market

The basic events are quotes and trades.

A **quote** shows the current prices at which people are willing to buy or sell an asset:

- **Bid:** the highest price someone offers to pay.
- **Ask:** the lowest price someone accepts to sell for.

For example, an Apple quote might show a bid of $200.10 and an ask of $200.12. The difference between them is called the **spread**.

A **trade** is an actual completed purchase or sale. For example, 100 Apple shares may trade at $200.11.

A **tick** is a general name for one small market update, usually a quote update or a trade. During busy periods, an actively traded asset can produce many ticks every second.

## Turning ticks into chart bars

Charts often do not show every trade separately. Instead, they group trades into a time window, such as one minute, five minutes, or one day. This summary is called a **bar** or **candlestick**.

A one-minute bar records:

- **Open:** the first trade price in that minute.
- **High:** the highest trade price.
- **Low:** the lowest trade price.
- **Close:** the final trade price.
- **Volume:** the number of shares, contracts, or coins traded.

These summaries make it possible to understand price movement without displaying every individual market event.

## Where the data comes from

Market data comes from exchanges, the organizations where assets are traded. In US equities, data can be consolidated through a **Securities Information Processor (SIP)**. **OPRA** distributes US options data. Cryptocurrency exchanges commonly provide market data through WebSocket connections.

A real-time market-data product might power a live price page, an investing app watchlist, a trading-platform candlestick chart, a price alert, or an automated trading strategy.

The challenge is volume and speed. A system may receive more than 100,000 updates per second while thousands of people expect their screens to update almost immediately. The rest of this article will design a system that receives raw exchange data, translates it into a consistent format, builds chart bars, and sends each user the updates they requested.

## Define what the system must do

Before choosing technologies, turn the problem into concrete requirements. For a 40-minute system-design interview, keep this list intentionally small: it defines the core flow without trying to design every production feature.

### Functional requirements

The system should:

- Ingest real-time quotes and trades from the supplied market-data feeds.
- Normalize each feed into one internal event format with a canonical instrument ID and event timestamp.
- Let a client subscribe to quotes, trades, and one-minute OHLCV bars for selected instruments over WebSocket. OHLCV means open, high, low, close, and volume.
- Build and publish the current one-minute bar from incoming trades.
- Detect feed disconnects or missing sequence numbers, then recover with a snapshot or resynchronization when the feed supports it.

### Non-functional requirements

The system should also meet a few measurable operational goals:

| Area | Example target |
| --- | --- |
| Throughput | Sustain 100,000 incoming updates per second, including short bursts. |
| Connections | Support thousands of concurrent WebSocket clients. |
| Internal latency | Process and route a normal event in under one millisecond within one region. |
| Slow clients | Keep client queues bounded; send the latest quote rather than every outdated quote. |
| Ordering | Preserve event order for one instrument where the feed provides it. |

The one-millisecond goal applies to the service's internal work, not necessarily to a person's screen. Network distance, browser scheduling, and connection quality can add noticeable delay for a user on the public internet.

Features such as market-data licensing, detailed historical storage, multi-region deployment, and late-bar corrections are important production concerns, but are reasonable follow-up discussion topics if time remains.

## Choose a design that fits the requirements

There is no single mandatory architecture. The right choice depends on how much traffic the system must handle, how quickly it must respond, and how much operational complexity the team can support. Here are three reasonable approaches.

### Option 1: One application does everything

One service connects to the feeds, normalizes updates, builds bars, and writes directly to every connected WebSocket client.

| Advantages | Trade-offs |
| --- | --- |
| Fastest to build and easiest for a small team to understand. | One process becomes a bottleneck as feeds and clients grow. |
| No extra broker or distributed coordination is required. | A slow client can consume memory or delay work unless queues are carefully bounded. |
| Good starting point for a prototype or a small internal tool. | Restarting the service interrupts both feed processing and all client connections. |

This option can satisfy the functional requirements at low volume. It is a poor long-term fit for the 100,000-updates-per-second target because every concern competes for the same CPU and memory.

### Option 2: Put a message broker in the middle

Feed handlers normalize events and publish them to a broker. Separate workers consume those events to build bars and distribute updates to clients. A broker is software that receives messages from one service and makes them available to other services.

| Advantages | Trade-offs |
| --- | --- |
| Separates ingest, bar building, and client delivery so each can scale independently. | Adds operational work: topics, partitions, retention, monitoring, and failure recovery. |
| A durable broker log can replay events after a worker restart. | A broker adds hops, which can increase latency. |
| Consumers can be added later for storage, alerts, or analytics. | Ordering is normally guaranteed only within a partition, not across all instruments. |

Kafka is a common choice when durable replay is the main goal. A lighter broker such as NATS can be attractive when low-latency delivery is more important than retaining every message. In either case, clients should connect to dedicated WebSocket workers rather than directly to the broker.

### Option 3: Partition by instrument and use edge fan-out workers

Route all updates for one instrument, such as `AAPL`, to the same partition worker. That worker owns the current quote and one-minute bar for its assigned instruments. It publishes updates to edge fan-out workers, which manage client WebSocket connections and subscriptions.

| Advantages | Trade-offs |
| --- | --- |
| Preserves a useful order for each instrument without a global lock. | Partition assignment and rebalancing are more complex than a single service. |
| Scales by adding workers and assigning more instrument partitions. | A heavily traded symbol can make one partition hot and may need special handling. |
| Keeps slow-client queues away from feed processing and bar aggregation. | Requires coordination to route an update only to edge workers with interested clients. |

This is the strongest fit for the stated throughput, latency, and slow-client requirements. The partition worker should be a single writer for its instruments: it updates their state without competing with another worker. Edge workers apply the slow-client policy by keeping only the newest quote for an instrument when a client falls behind.

## Recommended interview design

Start with Option 1. It is enough to explain the core functional requirements and gives the interviewer a concrete baseline:

```text
market feed -> one application -> WebSocket clients
                 |
                 -> one-minute bar aggregation
```

The application receives a market update, normalizes it, updates the current one-minute bar, and sends it to subscribed clients. For a prototype or a small internal tool, this is often the right design because it is easy to build and operate.

Then use the stated scale targets to expose its bottlenecks:

- Ingesting market data, updating bars, and writing to WebSocket connections compete for the same process's CPU.
- Slow clients can make outbound queues grow and consume the application's memory.
- One process has a practical throughput limit and a restart disconnects every client.

Only after naming those limits should the design evolve to Option 3:

```text
feed handlers -> instrument partitions -> edge fan-out workers -> WebSocket clients
                        |
                        -> one-minute bar aggregation
```

Partitions divide instruments among workers and preserve a useful order for each instrument. Edge fan-out workers own client connections, so their bounded queues and quote coalescing do not slow ingestion or bar aggregation.

A durable event log is a later addition for replay and recovery, not a required component of the first diagram. When it becomes necessary, keep it beside the immediate delivery path:

- The **hot path** processes a market update in memory and sends it to interested clients with minimal delay.
- The **durable path** stores events asynchronously for recovery or historical features.

The trade-off is deliberate: the newest live update can reach a client before it has been durably stored, but a slow disk or broker does not delay every market update.
