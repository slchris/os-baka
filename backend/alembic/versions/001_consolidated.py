"""Consolidated migration

Revision ID: 001_consolidated
Revises: 
Create Date: 2025-11-19

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.sql import func


# revision identifiers, used by Alembic.
revision = '001_consolidated'
down_revision = None
branch_labels = None
depends_on = None


def upgrade():
    # from 001_initial.py
    op.create_table(
        'assets',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('hostname', sa.String(length=255), nullable=False),
        sa.Column('ip_address', sa.String(length=45), nullable=False),
        sa.Column('mac_address', sa.String(length=17), nullable=False),
        sa.Column('asset_tag', sa.String(length=100), nullable=False),
        sa.Column('status', sa.String(length=50), nullable=False),
        sa.Column('os_type', sa.String(length=50), nullable=True),
        sa.Column('encryption_enabled', sa.Boolean(), nullable=True),
        sa.Column('last_seen', sa.DateTime(timezone=True), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=True),
        sa.Column('updated_at', sa.DateTime(timezone=True), nullable=True),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_assets_id'), 'assets', ['id'], unique=False)
    op.create_index(op.f('ix_assets_hostname'), 'assets', ['hostname'], unique=True)
    op.create_index(op.f('ix_assets_ip_address'), 'assets', ['ip_address'], unique=True)
    op.create_index(op.f('ix_assets_mac_address'), 'assets', ['mac_address'], unique=True)
    op.create_index(op.f('ix_assets_asset_tag'), 'assets', ['asset_tag'], unique=True)

    # from 002_add_users.py
    op.create_table(
        'users',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('username', sa.String(length=50), nullable=False),
        sa.Column('email', sa.String(length=100), nullable=False),
        sa.Column('hashed_password', sa.String(length=255), nullable=False),
        sa.Column('full_name', sa.String(length=100), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True, server_default='true'),
        sa.Column('is_superuser', sa.Boolean(), nullable=True, server_default='false'),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=func.now(), nullable=True),
        sa.Column('updated_at', sa.DateTime(timezone=True), onupdate=func.now(), nullable=True),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_users_id'), 'users', ['id'], unique=False)
    op.create_index(op.f('ix_users_username'), 'users', ['username'], unique=True)
    op.create_index(op.f('ix_users_email'), 'users', ['email'], unique=True)

    # from cd8cd7f982af_add_pxe_tables.py
    op.create_table(
        'pxe_configs',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('hostname', sa.String(length=255), nullable=False),
        sa.Column('mac_address', sa.String(length=17), nullable=False),
        sa.Column('ip_address', sa.String(length=15), nullable=False),
        sa.Column('netmask', sa.String(length=15), nullable=False, server_default='255.255.255.0'),
        sa.Column('gateway', sa.String(length=15), nullable=True),
        sa.Column('dns_servers', sa.String(length=255), nullable=True),
        sa.Column('boot_image', sa.String(length=255), nullable=True),
        sa.Column('boot_params', sa.Text(), nullable=True),
        sa.Column('os_type', sa.String(length=50), nullable=False, server_default='ubuntu'),
        sa.Column('enabled', sa.Boolean(), nullable=True, server_default='true'),
        sa.Column('deployed', sa.Boolean(), nullable=True, server_default='false'),
        sa.Column('last_boot', sa.DateTime(timezone=True), nullable=True),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('asset_id', sa.Integer(), nullable=True),
        sa.ForeignKeyConstraint(['asset_id'], ['assets.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_pxe_configs_hostname'), 'pxe_configs', ['hostname'], unique=True)
    op.create_index(op.f('ix_pxe_configs_mac_address'), 'pxe_configs', ['mac_address'], unique=True)
    
    op.create_table(
        'pxe_deployments',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('pxe_config_id', sa.Integer(), nullable=False),
        sa.Column('started_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('completed_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('status', sa.String(length=50), nullable=False, server_default='pending'),
        sa.Column('log_output', sa.Text(), nullable=True),
        sa.Column('error_message', sa.Text(), nullable=True),
        sa.Column('initiated_by', sa.Integer(), nullable=True),
        sa.ForeignKeyConstraint(['initiated_by'], ['users.id'], ),
        sa.ForeignKeyConstraint(['pxe_config_id'], ['pxe_configs.id'], ),
        sa.PrimaryKeyConstraint('id')
    )

    # from e93d6e7cbbdc_add_pxe_and_settings_tables.py
    op.create_table('system_settings',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('key', sa.String(length=255), nullable=False),
        sa.Column('value', sa.Text(), nullable=True),
        sa.Column('value_type', sa.String(length=50), nullable=False),
        sa.Column('category', sa.String(length=100), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('is_public', sa.Boolean(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_system_settings_id'), 'system_settings', ['id'], unique=False)
    op.create_index(op.f('ix_system_settings_key'), 'system_settings', ['key'], unique=True)
    op.create_index(op.f('ix_pxe_configs_id'), 'pxe_configs', ['id'], unique=False)
    op.create_unique_constraint(None, 'pxe_configs', ['ip_address'])
    op.create_index(op.f('ix_pxe_deployments_id'), 'pxe_deployments', ['id'], unique=False)

    # from 54fb642d31c9_add_pxe_service_config.py
    op.create_table('pxe_service_config',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('server_ip', sa.String(length=15), nullable=False),
        sa.Column('dhcp_range_start', sa.String(length=15), nullable=False),
        sa.Column('dhcp_range_end', sa.String(length=15), nullable=False),
        sa.Column('netmask', sa.String(length=15), nullable=False),
        sa.Column('gateway', sa.String(length=15), nullable=True),
        sa.Column('dns_servers', sa.String(length=255), nullable=True),
        sa.Column('tftp_root', sa.String(length=255), nullable=False),
        sa.Column('enable_bios', sa.Boolean(), nullable=False),
        sa.Column('enable_uefi', sa.Boolean(), nullable=False),
        sa.Column('bios_boot_file', sa.String(length=255), nullable=True),
        sa.Column('uefi_boot_file', sa.String(length=255), nullable=True),
        sa.Column('os_type', sa.String(length=50), nullable=False),
        sa.Column('debian_version', sa.String(length=50), nullable=False),
        sa.Column('debian_mirror', sa.String(length=255), nullable=True),
        sa.Column('enable_encrypted', sa.Boolean(), nullable=False),
        sa.Column('enable_unencrypted', sa.Boolean(), nullable=False),
        sa.Column('default_encryption', sa.Boolean(), nullable=False),
        sa.Column('luks_password', sa.String(length=255), nullable=True),
        sa.Column('preseed_template', sa.Text(), nullable=True),
        sa.Column('root_password', sa.String(length=255), nullable=True),
        sa.Column('default_username', sa.String(length=100), nullable=True),
        sa.Column('default_user_password', sa.String(length=255), nullable=True),
        sa.Column('container_name', sa.String(length=100), nullable=False),
        sa.Column('container_image', sa.String(length=255), nullable=False),
        sa.Column('container_status', sa.String(length=50), nullable=True),
        sa.Column('container_id', sa.String(length=255), nullable=True),
        sa.Column('service_enabled', sa.Boolean(), nullable=False),
        sa.Column('last_started', sa.DateTime(timezone=True), nullable=True),
        sa.Column('last_stopped', sa.DateTime(timezone=True), nullable=True),
        sa.Column('extra_config', sa.JSON(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_pxe_service_config_id'), 'pxe_service_config', ['id'], unique=False)

    # from 37f10934827a_add_usbkey_encryption_tables.py
    op.create_table('usb_keys',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('uuid', sa.String(length=36), nullable=False),
        sa.Column('label', sa.String(length=100), nullable=False),
        sa.Column('serial_number', sa.String(length=100), nullable=True),
        sa.Column('key_material', sa.Text(), nullable=False),
        sa.Column('salt', sa.String(length=64), nullable=False),
        sa.Column('iterations', sa.Integer(), nullable=True),
        sa.Column('backup_key', sa.Text(), nullable=True),
        sa.Column('recovery_code', sa.String(length=64), nullable=True),
        sa.Column('backup_count', sa.Integer(), nullable=True),
        sa.Column('last_backup_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('status', sa.Enum('PENDING', 'ACTIVE', 'BOUND', 'REVOKED', 'LOST', 'BACKUP', name='usbkeystatus'), nullable=False),
        sa.Column('is_initialized', sa.Boolean(), nullable=False),
        sa.Column('asset_id', sa.Integer(), nullable=True),
        sa.Column('bound_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('last_used_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('use_count', sa.Integer(), nullable=True),
        sa.Column('failed_attempts', sa.Integer(), nullable=True),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('revoked_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('revoked_by', sa.Integer(), nullable=True),
        sa.ForeignKeyConstraint(['asset_id'], ['assets.id'], ),
        sa.ForeignKeyConstraint(['created_by'], ['users.id'], ),
        sa.ForeignKeyConstraint(['revoked_by'], ['users.id'], ),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('asset_id'),
        sa.UniqueConstraint('serial_number')
    )
    op.create_index(op.f('ix_usb_keys_id'), 'usb_keys', ['id'], unique=False)
    op.create_index(op.f('ix_usb_keys_uuid'), 'usb_keys', ['uuid'], unique=True)
    op.create_table('usb_key_backups',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('usb_key_id', sa.Integer(), nullable=False),
        sa.Column('backup_uuid', sa.String(length=36), nullable=False),
        sa.Column('encrypted_backup', sa.Text(), nullable=False),
        sa.Column('backup_salt', sa.String(length=64), nullable=False),
        sa.Column('backup_type', sa.String(length=50), nullable=True),
        sa.Column('backup_size', sa.Integer(), nullable=True),
        sa.Column('checksum', sa.String(length=64), nullable=False),
        sa.Column('recovery_password', sa.String(length=255), nullable=True),
        sa.Column('is_recoverable', sa.Boolean(), nullable=True),
        sa.Column('recovered_count', sa.Integer(), nullable=True),
        sa.Column('last_recovered_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('expires_at', sa.DateTime(timezone=True), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['users.id'], ),
        sa.ForeignKeyConstraint(['usb_key_id'], ['usb_keys.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_usb_key_backups_backup_uuid'), 'usb_key_backups', ['backup_uuid'], unique=True)
    op.create_index(op.f('ix_usb_key_backups_id'), 'usb_key_backups', ['id'], unique=False)
    op.create_table('encryption_configs',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('asset_id', sa.Integer(), nullable=False),
        sa.Column('encryption_level', sa.Enum('TPM_ONLY', 'TPM_USBKEY', 'TPM_USBKEY_PASSWORD', 'USBKEY_ONLY', 'USBKEY_PASSWORD', 'PASSWORD_ONLY', name='encryptionlevel'), nullable=False),
        sa.Column('enable_tpm', sa.Boolean(), nullable=False),
        sa.Column('tpm_version', sa.String(length=10), nullable=True),
        sa.Column('tpm_pcr_policy', sa.Text(), nullable=True),
        sa.Column('tpm_sealed_key', sa.Text(), nullable=True),
        sa.Column('enable_usbkey', sa.Boolean(), nullable=False),
        sa.Column('usb_key_id', sa.Integer(), nullable=True),
        sa.Column('require_usbkey', sa.Boolean(), nullable=True),
        sa.Column('enable_password', sa.Boolean(), nullable=False),
        sa.Column('password_hash', sa.String(length=255), nullable=True),
        sa.Column('password_hint', sa.String(length=255), nullable=True),
        sa.Column('luks_version', sa.String(length=10), nullable=False),
        sa.Column('cipher', sa.String(length=50), nullable=True),
        sa.Column('key_size', sa.Integer(), nullable=True),
        sa.Column('hash_algorithm', sa.String(length=50), nullable=True),
        sa.Column('tpm_key_slot', sa.Integer(), nullable=True),
        sa.Column('usbkey_slot', sa.Integer(), nullable=True),
        sa.Column('password_slot', sa.Integer(), nullable=True),
        sa.Column('recovery_key', sa.Text(), nullable=True),
        sa.Column('recovery_key_slot', sa.Integer(), nullable=True),
        sa.Column('is_encrypted', sa.Boolean(), nullable=False),
        sa.Column('encryption_initialized', sa.Boolean(), nullable=False),
        sa.Column('last_unlocked_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('unlock_count', sa.Integer(), nullable=True),
        sa.Column('deployed_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('deployment_id', sa.Integer(), nullable=True),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
        sa.ForeignKeyConstraint(['asset_id'], ['assets.id'], ),
        sa.ForeignKeyConstraint(['deployment_id'], ['pxe_deployments.id'], ),
        sa.ForeignKeyConstraint(['usb_key_id'], ['usb_keys.id'], ),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('asset_id')
    )
    op.create_index(op.f('ix_encryption_configs_id'), 'encryption_configs', ['id'], unique=False)

def downgrade():
    # from 37f10934827a_add_usbkey_encryption_tables.py
    op.drop_index(op.f('ix_encryption_configs_id'), table_name='encryption_configs')
    op.drop_table('encryption_configs')
    op.drop_index(op.f('ix_usb_key_backups_id'), table_name='usb_key_backups')
    op.drop_index(op.f('ix_usb_key_backups_backup_uuid'), table_name='usb_key_backups')
    op.drop_table('usb_key_backups')
    op.drop_index(op.f('ix_usb_keys_uuid'), table_name='usb_keys')
    op.drop_index(op.f('ix_usb_keys_id'), table_name='usb_keys')
    op.drop_table('usb_keys')

    # from 54fb642d31c9_add_pxe_service_config.py
    op.drop_index(op.f('ix_pxe_service_config_id'), table_name='pxe_service_config')
    op.drop_table('pxe_service_config')

    # from e93d6e7cbbdc_add_pxe_and_settings_tables.py
    op.drop_index(op.f('ix_pxe_deployments_id'), table_name='pxe_deployments')
    op.drop_constraint(None, 'pxe_configs', type_='unique')
    op.drop_index(op.f('ix_pxe_configs_id'), table_name='pxe_configs')
    op.create_index('ix_pxe_configs_ip_address', 'pxe_configs', ['ip_address'], unique=False)
    op.drop_index(op.f('ix_system_settings_key'), table_name='system_settings')
    op.drop_index(op.f('ix_system_settings_id'), table_name='system_settings')
    op.drop_table('system_settings')

    # from cd8cd7f982af_add_pxe_tables.py
    op.drop_table('pxe_deployments')
    op.drop_index(op.f('ix_pxe_configs_mac_address'), table_name='pxe_configs')
    op.drop_index(op.f('ix_pxe_configs_ip_address'), table_name='pxe_configs')
    op.drop_index(op.f('ix_pxe_configs_hostname'), table_name='pxe_configs')
    op.drop_table('pxe_configs')

    # from 002_add_users.py
    op.drop_index(op.f('ix_users_email'), table_name='users')
    op.drop_index(op.f('ix_users_username'), table_name='users')
    op.drop_index(op.f('ix_users_id'), table_name='users')
    op.drop_table('users')

    # from 001_initial.py
    op.drop_index(op.f('ix_assets_asset_tag'), table_name='assets')
    op.drop_index(op.f('ix_assets_mac_address'), table_name='assets')
    op.drop_index(op.f('ix_assets_ip_address'), table_name='assets')
    op.drop_index(op.f('ix_assets_hostname'), table_name='assets')
    op.drop_index(op.f('ix_assets_id'), table_name='assets')
    op.drop_table('assets')
