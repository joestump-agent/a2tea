---
id: intro
title: Introduction
sidebar_label: Introduction
slug: /intro
description: a2tea is the bridge that lets an AI agent drive a terminal UI with A2UI.
---

# Introduction

a2tea is the bridge that lets an AI agent drive a terminal UI with A2UI. It
recognizes A2UI in a model's reply and draws it — instead of dumping raw JSON.

:::note Early, but real.
Rendering works for the core catalog: all five input components are editable,
modals open and close, and `ChildList` templates expand from the data model.
The main remaining gap is interactive tab switching. See the roadmap in the
[README](https://github.com/joestump-agent/a2tea#roadmap).
:::

## What is a2tea

a2tea parses the A2UI messages an agent emits — interleaved with conversational
text in an LLM response — and renders the described surfaces as Bubble Tea
models. The consumer is Joe's fork of
[charmbracelet/crush](https://github.com/charmbracelet/crush): a2tea is the
bridge that lets crush spot A2UI in a reply and render it as a live surface.

## Why a bridge

A host should not hand-roll detection of A2UI payloads in model output. It calls
[`Scan`](./api-reference.md) (or the cheap [`Contains`](./api-reference.md)
probe) and gets back the text and the typed A2UI messages, using the real A2UI
wire format — then hands the messages to [`Render`](./api-reference.md) for an
embeddable surface.

## The two standards

### A2UI · Google

A declarative protocol for agent-driven UIs. Agents send component descriptions
— not code — that clients render with their own widgets. a2tea targets v0.9.

### Bubble Tea · charm.land

The Elm Architecture, in Go: a model, an update function, a view. Styled with
Lip Gloss. a2tea renders A2UI surfaces as embeddable `tea.Model` values.
