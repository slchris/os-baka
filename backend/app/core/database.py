from sqlalchemy import create_engine
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from app.core.config import settings
import time
import logging

# Create database engine
engine = create_engine(
    settings.DATABASE_URL,
    pool_pre_ping=True,
    echo=True,  # Set to False in production
)

# Create session factory
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

# Create base class for models
Base = declarative_base()


def get_db():
    """Database dependency for FastAPI"""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

def wait_for_db():
    """Wait for the database to be ready"""
    logging.info("Waiting for database...")
    retries = 10
    while retries > 0:
        try:
            engine.connect()
            logging.info("Database is ready!")
            return
        except Exception as e:
            logging.warning(f"Database not ready, retrying... ({retries} retries left)")
            retries -= 1
            time.sleep(5)
    logging.error("Could not connect to database after multiple retries. Exiting.")
    exit(1)
