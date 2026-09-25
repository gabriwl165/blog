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
