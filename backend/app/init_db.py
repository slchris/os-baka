"""
Database initialization script
Creates default admin user and performs other setup tasks
"""
from sqlalchemy.orm import Session
from app.core.database import SessionLocal, engine, Base, wait_for_db
from app.core.security import get_password_hash
from app.models.user import User
from app.models.asset import Asset


def init_db(db: Session) -> None:
    """Initialize database with default data"""
    
    # Create tables
    Base.metadata.create_all(bind=engine)
    
    # Check if admin user exists
    admin = db.query(User).filter(User.username == "admin").first()
    
    if not admin:
        # Create default admin user
        admin = User(
            username="admin",
            email="admin@example.com",
            full_name="系统管理员",
            hashed_password=get_password_hash("admin123"),
            is_active=True,
            is_superuser=True
        )
        db.add(admin)
        db.commit()
        db.refresh(admin)
        print("✅ 创建默认管理员账户: admin / admin123")
    else:
        print("ℹ️  管理员账户已存在")
    
    print("✅ 数据库初始化完成")


def main() -> None:
    """Main function to run initialization"""
    print("🚀 开始初始化数据库...")
    
    # Wait for the database to be ready
    wait_for_db()

    db = SessionLocal()
    try:
        init_db(db)
    finally:
        db.close()
    
    print("🎉 所有初始化任务完成!")


if __name__ == "__main__":
    main()
