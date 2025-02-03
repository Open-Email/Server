package main

type contextKey string

const randomNonceContextKey = contextKey("random-nonce")
const signingFingerprintContextKey = contextKey("signing-fingerprint")
const userContextKey = contextKey("user")
const domainContextKey = contextKey("domain")
const linkContextKey = contextKey("link")
