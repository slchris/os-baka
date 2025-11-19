# Project Overview

This project is a PXE (Preboot Execution Environment) service and physical asset management system. It features a modern, Apple-style user interface for managing network booting of operating systems (macOS, Linux, Windows) and for tracking physical assets (IP addresses, MAC addresses, asset tags, etc.).

The project is a web application with a client-server architecture:

*   **Backend:** A RESTful API built with Python and the [FastAPI](https://fastapi.tiangolo.com/) framework. It uses [SQLAlchemy](https://www.sqlalchemy.org/) as an ORM to interact with a [PostgreSQL](https://www.postgresql.org/) database. [Alembic](https://alembic.sqlalchemy.org/en/latest/) is used for database migrations. Asynchronous tasks are handled by [Celery](https://docs.celeryq.dev/en/stable/) with [Redis](https://redis.io/) as the message broker.

*   **Frontend:** A single-page application built with the [Svelte](https://svelte.dev/) framework and [TypeScript](https://www.typescriptlang.org/). It uses [Vite](https://vitejs.dev/) for a fast development experience and [Axios](https://axios-http.com/) for making HTTP requests to the backend. The UI is styled with [SCSS](https://sass-lang.com/).

*   **Infrastructure:** The application is designed to be deployed using [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/). A `docker-compose.yml` file is provided to orchestrate the backend, frontend, PostgreSQL database, and Redis cache. [Nginx](https://www.nginx.com/) is used as a reverse proxy and [dnsmasq](http://www.thekelleys.org.uk/dnsmasq/doc.html) for DHCP and TFTP services for PXE booting.

# Building and Running

The recommended way to run the project is with Docker Compose.

## Using Docker Compose

1.  **Start all services:**
    ```bash
    cd docker
    docker-compose up -d
    ```

2.  **View logs:**
    ```bash
    docker-compose logs -f
    ```

The services will be accessible at the following URLs:

*   **Frontend:** http://localhost:5173
*   **Backend API:** http://localhost:8000
*   **API Docs:** http://localhost:8000/api/docs

## Local Development

### Backend

1.  **Navigate to the backend directory:**
    ```bash
    cd backend
    ```

2.  **Create a virtual environment:**
    ```bash
    python3 -m venv venv
    source venv/bin/activate
    ```

3.  **Install dependencies:**
    ```bash
    pip install -r requirements.txt
    ```

4.  **Configure environment variables:**
    ```bash
    cp .env.example .env
    # Edit .env to configure the database, etc.
    ```

5.  **Run database migrations:**
    ```bash
    alembic upgrade head
    ```

6.  **Start the development server:**
    ```bash
    uvicorn app.main:app --reload
    ```

### Frontend

1.  **Navigate to the frontend directory:**
    ```bash
    cd frontend
    ```

2.  **Install dependencies:**
    ```bash
    npm install
    ```

3.  **Start the development server:**
    ```bash
    npm run dev
    ```

# Development Conventions

*   **Backend:** The backend follows the standard FastAPI project structure, with code organized into `api`, `core`, `models`, and `schemas` modules. Database migrations are managed with Alembic.
*   **Frontend:** The frontend uses Svelte and TypeScript. Code is organized into `components`, `pages`, `lib`, and `styles`.
*   **Git:** The project uses Git for version control. It is hosted on GitHub. (Based on the `.git` directory and `readme.md` content).
*   **Testing:** TODO: No testing framework or configurations were found. It's recommended to add unit and integration tests for both the backend and frontend.
