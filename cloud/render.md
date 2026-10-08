# Lab 10 — Render Deployment

## Platform

Render Free Web Service

## Source

Existing container image:

ghcr.io/abdullohml/devops-intro/quicknotes:v0.1.0

Using the existing image ensures that Render runs the same artifact built by CI and scanned in Lab 9.

## Configuration

- Service name: quicknotes-lab10
- Region: Frankfurt (EU Central)
- Instance type: Free
- PORT=10000
- ADDR=:10000
- Health check path: /health

## Public URL

https://quicknotes-lab10-0x46.onrender.com

## Deploy evidence

Render startup log:

quicknotes listening on :10000 (notes loaded: 0)
Your service is live
