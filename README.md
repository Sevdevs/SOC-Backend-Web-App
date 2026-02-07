# Incident Tracker Web App

A Go web app that helps you practice incident triage workflows. It includes a
simple in-memory API, an incident queue, and a detail view for notes, severity,
and status changes.

## Features
- Go HTTP server with an in-memory CRUD API
- Incident queue with severity, status, owner, tags, and IOCs
- Detail view for updates and investigation notes
- Search and filter controls for triage workflows
- Responsive layout optimized for desktop and mobile

## Getting Started
1. Ensure Go 1.22+ is installed.
2. Run the server:
   go run .
3. Open your browser and visit:
   localhost:8080

## Notes
- Data is stored in memory and resets when the server restarts.
- Replace the mock store with a database when you want persistence.
