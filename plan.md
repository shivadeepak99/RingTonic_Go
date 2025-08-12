**RingTonic MVP: System Architecture & Design**
Version: 1.
Status: Design Finalized for MVP

**1. Introduction**
This document outlines the complete technical architecture for the RingTonic MVP. Its
purpose is to provide a clear, shared understanding of how each component interacts,
what its specific responsibilities are, and what is required to build it. We will visualize
the system as a restaurant kitchen to make the roles and data flow intuitive.
**2. Component Responsibility Table**
Think of our system as a restaurant. Each component has one specific job it does
really well.
    **Component Role in this**
       **Project (What**
       **it does)**
          **Input Output Trigger**
    **Next.js (Web**
    **UI)**
       **The "Waiter"**
       **for Web Users.**
       Presents the
       menu, takes the
       link, and
       provides the
       download link.
          User-pasted
          URL string.
             POST request to
             Go Backend
             with URL in
             JSON payload.
                User clicks
                "Create
                Ringtone"
                button.
    **Flutter (Mobile**
    **App)**
       **The "Waiter"**
       **for Mobile**
       **Users.** Same as
       web UI, plus
       ability to set
       ringtone
       directly.
          User-pasted
          URL string.
             1. POST request
             to Go
             Backend.<br>2.
             System call to
             Android's
             RingtoneManag
             er.
                1. User clicks
                "Create
                Ringtone".<br>2.
                User clicks "Set
                as Ringtone".
    **Go Backend**
    **(API)**
       **The**
       **"Restaurant**
       **Manager."**
       Takes orders,
       creates a ticket
       (jobId), hands
       ticket to n8n,
       tracks status,
       manages
       download
          JSON requests
          from frontend,
          JSON callbacks
          from n8n.
             1. JSON
             response with
             jobId.<br>2.
             Webhook call to
             n8n.<br>3. Raw
             .mp3 file for
             download.
                Incoming HTTP
                request on an
                API endpoint.


```
pickup.
n8n
(Orchestrator)
The "Head
Chef." Receives
ticket from
manager, directs
yt-dlp and
FFmpeg,
ensures process
order.
Webhook call
from Go
Backend with
jobId and url.
```
1. Shell
commands to
yt-dlp and
FFmpeg.<br>2.
Final webhook
callback to Go
Backend with
job outcome.
    Webhook
    trigger hit by Go
    Backend.
**yt-dlp
(Extractor)
The "Ingredient
Prep Cook."**
Gets raw audio
from the video
link.
A video URL. A raw audio file
(e.g., .m4a,
.webm).
A shell
command
executed by
n8n.
**FFmpeg
(Processor)
The "Line
Cook."** Takes
raw audio and
converts/trims it
to .mp3.
A raw audio file
from yt-dlp.
A final, trimmed
.mp3 file.
A shell
command
executed by
n8n.
**SQLite
(Database)
The "Order
Logbook."**
Stores the order
(jobId), its
current status,
and the final
filename.
SQL queries
from the Go
Backend.
Rows of data
representing job
information.
The Go Backend
needs to create,
update, or read
a job's status.
**3. Step-by-Step Workflow (The End-to-End Journey)**
Let's trace a single user request from start to finish.
1. **User Action (Frontend):** A user pastes a video link into the Next.js or Flutter app
and clicks "Create". The app's UI immediately enters a "Processing..." state.
2. **Job Creation (Backend - Sync):** The frontend sends a POST request to the Go
Backend (/api/v1/create-job) with the URL. The backend:
○ Generates a unique jobId.
○ Creates a PENDING job entry in the SQLite database.
○ Immediately responds to the frontend with the { "jobId": "..." }. This is a quick,
synchronous action.
3. **Hand-off to Kitchen (Backend -> n8n - Async):** After responding, the Go
Backend makes a "fire-and-forget" POST request to the n8n webhook, sending


```
the { "url": "...", "jobId": "..." }.
```
4. **Processing the Order (n8n Workflow):** The n8n webhook triggers a sequence of
    nodes:
       ○ **Node 1 (yt-dlp):** Executes a shell command to download the audio from the
          URL, saving it to a temporary location.
       ○ **Node 2 (FFmpeg):** Executes a second shell command to trim the audio and
          convert it to a final .mp3 file, saving it to a persistent storage directory.
       ○ **Error Handling:** If either command fails, the workflow's error path is
          triggered.
5. **Status Polling (Frontend):** While the backend processes, the frontend app starts
    a timer. Every few seconds, it polls the Go Backend (GET
    /api/v1/jobs/{jobId}/status) to check on the order.
6. **Notifying Completion (n8n -> Backend - Async):** Once the FFmpeg node
    succeeds, n8n sends a final callback POST request to the Go Backend
    (/api/v1/jobs/callback) with the result: { "jobId": "...", "status": "COMPLETED",
    "filename": "{jobId}.mp3" }. If it failed, it sends a FAILED status with an error
    message.
7. **Updating the Logbook (Backend):** The Go Backend receives the callback and
    updates the job's status to COMPLETED or FAILED in the SQLite database.
8. **Displaying the Result (Frontend):** On its next poll, the frontend receives the
    COMPLETED status. It stops polling and updates the UI to show a "Download
    Ringtone" or "Set as Ringtone" button.
9. **Final Action (Download/Set):**
    ○ **Web:** Clicking "Download" makes a GET request to the Go Backend, which
       serves the .mp3 file from storage.
    ○ **Mobile (Flutter):** Clicking "Set as Ringtone" downloads the file, requests the
       necessary Android permissions, and uses the RingtoneManager to set the
       new ringtone.
**4. API Endpoints (The Backend "Menu")**
These are the public and internal "doors" to our Go Backend.
**Endpoint Method Description Target Audience**
/api/v1/create-job POST Initiates a new
ringtone generation
job.
Frontend
(Web/Mobile)
/api/v1/jobs/{jobId}/st
atus
GET Retrieves the current
status of a specific
Frontend
(Web/Mobile)


```
job.
/api/v1/download/{file
name}
GET Serves the completed
ringtone MP3 file for
download.
Frontend
(Web/Mobile)
/api/v1/jobs/callback POST Receives job
outcome notifications
from the processing
workflow.
Internal (n8n only)
```
**5. Core Skills Required**
    **Component Critical Skills for MVP**
    **Go Backend** Building a basic REST API (net/http), handling
       JSON, using database/sql with SQLite, making
       outbound HTTP requests, and serving static
       files.
    **n8n** Creating Webhook triggers, using the "Execute
       Command" node, using expressions to access
       data (e.g., {{ $json.body.jobId }}), and using the
       "HTTP Request" node for callbacks.
    **yt-dlp & FFmpeg** Knowing the basic command-line flags to
       extract audio (-x) and convert/trim (-i, -t, -c
       copy).
    **Next.js / Flutter** Core UI library fundamentals (React/Flutter),
       state management, making asynchronous API
       calls (fetch, dio), and handling
       promises/futures.
    **Flutter (Specific) Using MethodChannel to call native Android**
       **code.** This is the most complex and
       non-negotiable skill for the mobile component.
       Understanding the Android permission model is
       also crucial.
    **DevOps** Basic understanding of Docker and
       docker-compose to run the entire stack locally
       for development.
**6. Integration Points (The "Contracts")**


These are the seams of the system. They MUST be agreed upon before coding begins.

1. **Frontend <-> Go Backend:** The openapi.yaml specification is the master
    contract defining all public API paths, request/response bodies, and error codes.
2. **Go Backend -> n8n:** The exact n8n webhook URL and the JSON payload
    structure: { "url": string, "jobId": string }.
3. **n8n -> Go Backend:** The callback URL (/api/v1/jobs/callback) and its required
    JSON payload: { "jobId": string, "status": "COMPLETED" \| "FAILED", ... }.
4. **Flutter <-> Native Android:** The MethodChannel name (e.g.,
    com.ringtone.app/channel) and the exact method names to be called from Dart
    (e.g., setRingtone).
**7. Risk & Complexity Map
Risk Area Complexity Why it's Risky & How to
Mitigate
Android Ringtone Setting High** Android permissions
(WRITE_SETTINGS) and
Scoped Storage are complex
and vary by OS version.
**Mitigation:** Create a tiny,
separate Flutter prototype
app to solve this specific
problem first before
integrating it into the main
app.
**yt-dlp Reliability High** YouTube and others actively
change their sites, which can
break the downloader without
warning. **Mitigation:** Accept
this as a reality. Build robust
error handling in the n8n
workflow to catch failures
gracefully and report a FAILED
status to the user.
**Async State Management Medium** Keeping the UI and backend in
sync during a background
process can be tricky.
**Mitigation:** For the MVP, a
simple polling mechanism is
robust and sufficient. Avoid
over-engineering with


```
WebSockets until necessary.
```
**8. Considerations & Future Enhancements**
    ● **Scalability:** For production, move from the local filesystem to an object storage
       service like S3. This is the most critical step for scaling.
    ● **Security:** Secure the internal n8n callback endpoint with a shared secret key.
       Implement rate limiting on public APIs to prevent abuse.
    ● **User Experience:** For V2, provide more granular progress updates to the user
       (e.g., "Downloading," "Converting...").
    ● **Monitoring:** Implement centralized logging to easily trace a job's journey from
       start to finish across all components.


