-- Development seed aligned with web-pc/src/data/higoos.ts demo data.
BEGIN;

INSERT INTO users (id, username, display_name, email, status, locale, timezone)
VALUES
  ('00000000-0000-4000-8000-000000000001', 'admin', 'HiGoOS 管理员', 'admin@higoos.local', 'active', 'zh-CN', 'Asia/Shanghai'),
  ('00000000-0000-4000-8000-000000000002', 'family', '家庭成员', 'family@higoos.local', 'active', 'zh-CN', 'Asia/Shanghai'),
  ('00000000-0000-4000-8000-000000000003', 'project', '项目组成员', 'project@higoos.local', 'active', 'zh-CN', 'Asia/Shanghai')
ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name, email = EXCLUDED.email, status = EXCLUDED.status;

INSERT INTO groups (id, slug, name, description)
VALUES
  ('00000000-0000-4000-8000-000000000011', 'family', '家庭组', '家庭空间默认可见成员'),
  ('00000000-0000-4000-8000-000000000012', 'project-team', '项目组', '团队空间项目协作成员')
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description;

INSERT INTO group_members (group_id, user_id, role_label)
VALUES
  ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000001', 'owner'),
  ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000002', 'member'),
  ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000001', 'owner'),
  ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000003', 'member')
ON CONFLICT (group_id, user_id) DO UPDATE SET role_label = EXCLUDED.role_label;

INSERT INTO roles (id, slug, name, permissions)
VALUES
  ('00000000-0000-4000-8000-000000000021', 'admin', '系统管理员', '["*"]'),
  ('00000000-0000-4000-8000-000000000022', 'family-member', '家庭成员', '["files:read", "assistant:chat"]'),
  ('00000000-0000-4000-8000-000000000023', 'project-member', '项目成员', '["files:read", "files:write", "agent:preview"]')
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, permissions = EXCLUDED.permissions;

INSERT INTO user_roles (user_id, role_id)
VALUES
  ('00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000021'),
  ('00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000022'),
  ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000023')
ON CONFLICT (user_id, role_id) DO NOTHING;

INSERT INTO spaces (id, slug, name, description, owner_user_id, owner_group_id, quota_bytes)
VALUES
  ('10000000-0000-4000-8000-000000000001', 'home-space', '家庭空间', '家庭保险、保修、证件和生活知识库', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000011', 12000000000000),
  ('10000000-0000-4000-8000-000000000002', 'team-space', '团队空间', '项目资料、合同、会议纪要和素材', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000012', 8000000000000),
  ('10000000-0000-4000-8000-000000000003', 'photos-media', '照片与视频', '家庭相册、视频和自动生成回忆', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000011', 10000000000000),
  ('10000000-0000-4000-8000-000000000004', 'finance-receipts', '财务票据', '发票、票据、报销和归档规则', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000011', 2000000000000),
  ('10000000-0000-4000-8000-000000000005', 'project-materials', '项目资料', '资料包和知识图谱输出', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000012', 6000000000000),
  ('10000000-0000-4000-8000-000000000006', 'docker-data', 'Docker 数据', '容器持久化目录和 compose 配置', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000011', 4000000000000),
  ('10000000-0000-4000-8000-000000000007', 'backup-archive', '备份归档', '快照、异地备份和恢复点', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000011', 18000000000000)
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, quota_bytes = EXCLUDED.quota_bytes;

INSERT INTO dock_apps (id, name, icon_path, badge_count, is_utility, sort_order)
VALUES
  ('file-manager', '文件管理', 'assets/higoos-dock/icons/01-file-manager.png', 2, false, 1),
  ('storage-monitor', '存储管理', 'assets/higoos-dock/icons/02-storage-manager.png', NULL, false, 2),
  ('ai-file-steward', 'AI 文件管家', 'assets/higoos-dock/icons/03-ai-file-steward.png', 6, false, 3),
  ('agent-workbench', 'Agent 工作台', 'assets/higoos-dock/icons/04-agent-workbench.png', NULL, false, 4),
  ('ai-assistant', 'AI 助手', 'assets/higoos-dock/icons/05-ai-assistant.png', NULL, false, 5),
  ('backup-sync', '备份同步', 'assets/higoos-dock/icons/06-backup-sync.png', 1, false, 6),
  ('photo-media', '相册媒体', 'assets/higoos-dock/icons/07-photo-media.png', NULL, false, 7),
  ('download-center', '下载中心', 'assets/higoos-dock/icons/08-download-center.png', NULL, false, 8),
  ('app-center', '应用中心', 'assets/higoos-dock/icons/09-app-center.png', NULL, false, 9),
  ('docker', 'Docker', 'assets/higoos-dock/icons/10-docker.png', NULL, false, 10),
  ('security-center', '安全中心', 'assets/higoos-dock/icons/11-security-center.png', 3, false, 11),
  ('device-monitor', '设备监控', 'assets/higoos-dock/icons/12-device-monitor.png', NULL, false, 12),
  ('system-settings', '系统设置', 'assets/higoos-dock/icons/13-system-settings.png', NULL, false, 13),
  ('remote-access', '远程访问', 'assets/higoos-dock/icons/14-remote-access.png', NULL, false, 14)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, icon_path = EXCLUDED.icon_path, badge_count = EXCLUDED.badge_count, sort_order = EXCLUDED.sort_order;

INSERT INTO desktop_windows (id, app_id, title, subtitle, status_text, status_tone, x, y, width, height, z_index)
VALUES
  ('file-manager', 'file-manager', '文件管理', '家庭空间 / 团队空间 / 语义搜索', 'AI 索引已同步', 'green', 48, 92, 700, 560, 4),
  ('ai-file-steward', 'ai-file-steward', 'AI 文件管家', '智能整理 / 权限审计 / 回滚', '6 条建议', 'orange', 778, 108, 472, 520, 5),
  ('agent-workbench', 'agent-workbench', 'Agent 工作台', '工作流 / 工具权限 / 执行确认', '需要确认', 'blue', 548, 388, 690, 360, 6),
  ('storage-monitor', 'storage-monitor', '存储管理', '主机卷 / SMART / 容量', '后端同步', 'green', 960, 80, 360, 296, 3),
  ('photo-media', 'photo-media', '相册媒体', '时间线 / 人物地点 / 媒体转码', '回忆生成', 'blue', 118, 118, 760, 536, 7),
  ('download-center', 'download-center', '下载中心', 'BT / HTTP / 磁力 / 自动归档', '队列运行', 'green', 228, 136, 720, 500, 8),
  ('docker', 'docker', 'Docker', '容器 / Compose / 端口资源', '4 个运行', 'green', 260, 112, 760, 520, 9),
  ('security-center', 'security-center', '安全中心', '权限 / 风险 / 审计回滚', '3 条风险', 'red', 300, 126, 760, 530, 10),
  ('device-monitor', 'device-monitor', '设备监控', '性能趋势 / 告警 / 系统日志', '实时', 'green', 170, 104, 760, 520, 11),
  ('system-settings', 'system-settings', '系统设置', '网络 / 模型 / 隐私 / 更新', '已同步', 'blue', 210, 96, 760, 536, 12),
  ('remote-access', 'remote-access', '远程访问', '域名 / 穿透 / MFA / 分享检查', '安全', 'green', 248, 116, 740, 514, 13)
ON CONFLICT (id) DO UPDATE SET subtitle = EXCLUDED.subtitle, status_text = EXCLUDED.status_text, status_tone = EXCLUDED.status_tone, x = EXCLUDED.x, y = EXCLUDED.y, width = EXCLUDED.width, height = EXCLUDED.height, z_index = EXCLUDED.z_index;

INSERT INTO folder_acl (id, space_id, principal_group_id, path_prefix, effect, access)
VALUES
  ('11000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000011', '/', 'allow', 'read'),
  ('11000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000012', '/', 'allow', 'write'),
  ('11000000-0000-4000-8000-000000000003', '10000000-0000-4000-8000-000000000004', '00000000-0000-4000-8000-000000000011', '/', 'deny', 'write')
ON CONFLICT (id) DO UPDATE SET effect = EXCLUDED.effect, access = EXCLUDED.access;

INSERT INTO file_nodes (id, space_id, parent_id, name, node_kind, mime_type, size_bytes, path, permission_label, ai_summary, modified_at)
VALUES
  ('20000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', NULL, '家庭空间', 'folder', NULL, 0, '/', '家人可见', '', now()),
  ('20000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000002', NULL, '团队空间', 'folder', NULL, 0, '/', '项目组', '', now()),
  ('20000000-0000-4000-8000-000000000003', '10000000-0000-4000-8000-000000000003', NULL, '照片与视频', 'folder', NULL, 0, '/', '家人可见', '', now()),
  ('20000000-0000-4000-8000-000000000004', '10000000-0000-4000-8000-000000000004', NULL, '财务票据', 'folder', NULL, 0, '/', '仅管理员', '', now()),
  ('20000000-0000-4000-8000-000000000101', '10000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', '2026 家庭保险与保修资料.pdf', 'file', 'application/pdf', 18400000, '/2026 家庭保险与保修资料.pdf', '家人可见', '包含 6 份家电保修单，2 项将在 30 天内到期。', now() - interval '3 hours'),
  ('20000000-0000-4000-8000-000000000102', '10000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000002', '客户A_合同_最终版.docx', 'file', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 2800000, '/客户A_合同_最终版.docx', '项目组', '识别到付款节点和保密条款，建议加入项目资料图谱。', now() - interval '1 day'),
  ('20000000-0000-4000-8000-000000000103', '10000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000003', '五一旅行照片精选', 'album', NULL, 4200000000, '/五一旅行照片精选', '家人可见', '832 张照片，已筛出 124 张清晰照片和 18 段视频。', now() - interval '1 day'),
  ('20000000-0000-4000-8000-000000000104', '10000000-0000-4000-8000-000000000004', '20000000-0000-4000-8000-000000000004', '下载目录', 'folder', NULL, 642000000, '/下载目录', '仅管理员', '发现 31 张发票，其中 4 张可能重复，建议按年月归档。', now() - interval '2 days'),
  ('20000000-0000-4000-8000-000000000105', '10000000-0000-4000-8000-000000000004', '20000000-0000-4000-8000-000000000104', '未归档发票', 'folder', NULL, 642000000, '/下载目录/未归档发票', '仅管理员', '发现 31 张发票，其中 4 张可能重复，建议按年月归档。', now() - interval '2 days')
ON CONFLICT (space_id, path) DO UPDATE SET name = EXCLUDED.name, node_kind = EXCLUDED.node_kind, size_bytes = EXCLUDED.size_bytes, permission_label = EXCLUDED.permission_label, ai_summary = EXCLUDED.ai_summary, modified_at = EXCLUDED.modified_at;

INSERT INTO file_tags (id, name, color)
VALUES
  ('21000000-0000-4000-8000-000000000001', '保修', '#2563eb'),
  ('21000000-0000-4000-8000-000000000002', '家庭知识库', '#16a34a'),
  ('21000000-0000-4000-8000-000000000003', '需提醒', '#f97316'),
  ('21000000-0000-4000-8000-000000000004', '合同', '#7c3aed'),
  ('21000000-0000-4000-8000-000000000005', '客户A', '#0891b2'),
  ('21000000-0000-4000-8000-000000000006', '权限敏感', '#dc2626'),
  ('21000000-0000-4000-8000-000000000007', '旅行', '#0f766e'),
  ('21000000-0000-4000-8000-000000000008', '人物已识别', '#65a30d'),
  ('21000000-0000-4000-8000-000000000009', '可生成回忆', '#2563eb'),
  ('21000000-0000-4000-8000-000000000010', '待整理', '#ea580c'),
  ('21000000-0000-4000-8000-000000000011', '发票', '#b91c1c'),
  ('21000000-0000-4000-8000-000000000012', '重复项', '#9333ea')
ON CONFLICT (name) DO UPDATE SET color = EXCLUDED.color;

INSERT INTO file_node_tags (file_id, tag_id)
VALUES
  ('20000000-0000-4000-8000-000000000101', '21000000-0000-4000-8000-000000000001'),
  ('20000000-0000-4000-8000-000000000101', '21000000-0000-4000-8000-000000000002'),
  ('20000000-0000-4000-8000-000000000101', '21000000-0000-4000-8000-000000000003'),
  ('20000000-0000-4000-8000-000000000102', '21000000-0000-4000-8000-000000000004'),
  ('20000000-0000-4000-8000-000000000102', '21000000-0000-4000-8000-000000000005'),
  ('20000000-0000-4000-8000-000000000102', '21000000-0000-4000-8000-000000000006'),
  ('20000000-0000-4000-8000-000000000103', '21000000-0000-4000-8000-000000000007'),
  ('20000000-0000-4000-8000-000000000103', '21000000-0000-4000-8000-000000000008'),
  ('20000000-0000-4000-8000-000000000103', '21000000-0000-4000-8000-000000000009'),
  ('20000000-0000-4000-8000-000000000105', '21000000-0000-4000-8000-000000000010'),
  ('20000000-0000-4000-8000-000000000105', '21000000-0000-4000-8000-000000000011'),
  ('20000000-0000-4000-8000-000000000105', '21000000-0000-4000-8000-000000000012')
ON CONFLICT (file_id, tag_id) DO NOTHING;

INSERT INTO file_versions (id, file_id, version_no, storage_uri, size_bytes, created_by)
VALUES
  ('22000000-0000-4000-8000-000000000101', '20000000-0000-4000-8000-000000000101', 1, 'fixture://nas-root/home-space/insurance-warranty.md', 18400000, '00000000-0000-4000-8000-000000000001'),
  ('22000000-0000-4000-8000-000000000102', '20000000-0000-4000-8000-000000000102', 1, 'fixture://nas-root/team-space/customer-a-contract.md', 2800000, '00000000-0000-4000-8000-000000000003')
ON CONFLICT (file_id, version_no) DO UPDATE SET storage_uri = EXCLUDED.storage_uri, size_bytes = EXCLUDED.size_bytes;

INSERT INTO storage_pools (id, name, pool_type, used_percent, total_bytes, health, temperature_c)
VALUES
  ('30000000-0000-4000-8000-000000000001', '开发主机根卷', '主机卷', 10, 245107195904, 'healthy', 0),
  ('30000000-0000-4000-8000-000000000002', '开发主机数据卷', '主机卷', 41, 245107195904, 'healthy', 0),
  ('30000000-0000-4000-8000-000000000003', '开发主机外接卷', '主机卷', 93, 214958080, 'warning', 0)
ON CONFLICT (name) DO UPDATE SET pool_type = EXCLUDED.pool_type, used_percent = EXCLUDED.used_percent, total_bytes = EXCLUDED.total_bytes, health = EXCLUDED.health, temperature_c = EXCLUDED.temperature_c;

INSERT INTO disks (id, slot, serial_number, size_bytes, state, temperature_c, pool_id)
VALUES
  ('31000000-0000-4000-8000-000000000001', '1', '/dev/disk-root', 245107195904, 'healthy', 0, '30000000-0000-4000-8000-000000000001'),
  ('31000000-0000-4000-8000-000000000002', '2', '/dev/disk-data', 245107195904, 'healthy', 0, '30000000-0000-4000-8000-000000000002'),
  ('31000000-0000-4000-8000-000000000003', '3', '/dev/disk-external', 214958080, 'healthy', 0, '30000000-0000-4000-8000-000000000003')
ON CONFLICT (slot) DO UPDATE SET state = EXCLUDED.state, temperature_c = EXCLUDED.temperature_c, pool_id = EXCLUDED.pool_id;

INSERT INTO volumes (id, pool_id, name, mount_path, size_bytes, used_bytes, filesystem)
VALUES
  ('32000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', 'root', '/', 245107195904, 24510719590, 'apfs'),
  ('32000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000002', 'data', '/System/Volumes/Data', 245107195904, 100493950320, 'apfs'),
  ('32000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000003', 'external', '/Volumes/External', 214958080, 199911014, 'apfs')
ON CONFLICT (mount_path) DO UPDATE SET used_bytes = EXCLUDED.used_bytes, filesystem = EXCLUDED.filesystem;

INSERT INTO smart_reports (id, disk_id, health, temperature_c, attributes)
VALUES
  ('33000000-0000-4000-8000-000000000001', '31000000-0000-4000-8000-000000000001', 'healthy', 0, '{"source": "host_df", "filesystem_usage": 10}'),
  ('33000000-0000-4000-8000-000000000002', '31000000-0000-4000-8000-000000000003', 'warning', 0, '{"source": "host_df", "filesystem_usage": 93}')
ON CONFLICT (id) DO UPDATE SET health = EXCLUDED.health, temperature_c = EXCLUDED.temperature_c, attributes = EXCLUDED.attributes;

INSERT INTO ai_models (id, provider, model_key, display_name, runtime, context_tokens, metadata)
VALUES
  ('40000000-0000-4000-8000-000000000001', 'local', 'higo-embed-zh-1536', 'HiGo Embedding zh 1536', 'local', 8192, '{"purpose": "semantic-index"}'),
  ('40000000-0000-4000-8000-000000000002', 'local', 'higo-assistant-local', 'HiGo Assistant Local', 'local', 32768, '{"privacy": "private-space"}')
ON CONFLICT (provider, model_key) DO UPDATE SET display_name = EXCLUDED.display_name, runtime = EXCLUDED.runtime, context_tokens = EXCLUDED.context_tokens;

INSERT INTO model_policies (id, name, runtime, model_id, scope, enforce_local_for_private)
VALUES
  ('41000000-0000-4000-8000-000000000001', '隐私空间强制本地推理', 'local', '40000000-0000-4000-8000-000000000002', '{"spaces": ["home-space", "finance-receipts"]}', true)
ON CONFLICT (name) DO UPDATE SET runtime = EXCLUDED.runtime, model_id = EXCLUDED.model_id, scope = EXCLUDED.scope, enforce_local_for_private = EXCLUDED.enforce_local_for_private;

INSERT INTO privacy_policies (id, name, rules)
VALUES
  ('42000000-0000-4000-8000-000000000001', '默认隐私策略', '[{"match": "finance-receipts", "runtime": "local"}, {"match": "权限敏感", "share": "review"}]')
ON CONFLICT (name) DO UPDATE SET rules = EXCLUDED.rules;

INSERT INTO index_jobs (id, file_id, job_type, status, progress, completed_at)
VALUES
  ('43000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000101', 'semantic-index', 'succeeded', 100, now()),
  ('43000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000102', 'semantic-index', 'succeeded', 100, now())
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, progress = EXCLUDED.progress, completed_at = EXCLUDED.completed_at;

INSERT INTO document_chunks (id, file_id, chunk_no, content, token_count)
VALUES
  ('44000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000101', 0, '家庭保险与保修资料摘要：6 份家电保修单，2 项将在 30 天内到期。', 42),
  ('44000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000102', 0, '客户 A 合同最终版包含付款节点、保密条款和每日快照备份状态。', 39)
ON CONFLICT (file_id, chunk_no) DO UPDATE SET content = EXCLUDED.content, token_count = EXCLUDED.token_count;

INSERT INTO entity_links (id, file_id, entity_type, entity_value, confidence)
VALUES
  ('45000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000102', 'customer', '客户A', 0.9800),
  ('45000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000101', 'reminder', '保修到期', 0.9200)
ON CONFLICT (id) DO UPDATE SET confidence = EXCLUDED.confidence;

INSERT INTO knowledge_edges (id, source_entity, target_entity, relation, confidence)
VALUES
  ('46000000-0000-4000-8000-000000000001', '客户A', '客户A_合同_最终版.docx', 'owns_contract', 0.9900),
  ('46000000-0000-4000-8000-000000000002', '家庭保险', '保修到期', 'requires_reminder', 0.9100)
ON CONFLICT (source_entity, target_entity, relation) DO UPDATE SET confidence = EXCLUDED.confidence;

INSERT INTO risk_actions (id, actor_user_id, risk_level, title, detail, action_key, status)
VALUES
  ('50000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000001', 'medium', '下载目录智能整理', '31 张发票、12 个安装包和 4 个重复压缩包可按规则归档。', 'preview_archive_downloads', 'queued'),
  ('50000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', 'high', '过期分享链接', '发现 3 个公开链接仍可访问，包含团队空间资料。', 'review_share_permissions', 'queued'),
  ('50000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001', 'low', '相似照片清理', '五一旅行相册中有 86 张连拍相似照片，可保留清晰版本。', 'select_best_photos', 'queued')
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, detail = EXCLUDED.detail, risk_level = EXCLUDED.risk_level;

INSERT INTO audit_events (id, actor_user_id, event_type, target_type, target_id, summary, metadata)
VALUES
  ('51000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000001', 'ai_steward.read', 'path', '/下载/票据', '09:41 文件管家读取 /下载/票据，仅生成建议，未移动文件', '{"readonly": true}'),
  ('51000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', 'agent.reminder.create', 'agent', '家庭资料助手', '09:22 Agent 创建家庭保修提醒，等待管理员确认', '{"requires_confirmation": true}'),
  ('51000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001', 'rollback.rename', 'files', 'batch-rename-001', '昨天 18:36 撤销 12 个文件重命名，已恢复原路径', '{"restored": 12}')
ON CONFLICT (id) DO UPDATE SET summary = EXCLUDED.summary, metadata = EXCLUDED.metadata;

INSERT INTO security_findings (id, title, detail, risk_level, source, target_type, target_id, status)
VALUES
  ('52000000-0000-4000-8000-000000000001', '权限审计', '3 个分享链接建议收紧', 'medium', 'security-center', 'share_links', 'stale-public-links', 'queued')
ON CONFLICT (id) DO UPDATE SET detail = EXCLUDED.detail, risk_level = EXCLUDED.risk_level;

INSERT INTO metrics_samples (id, metric_key, label, value_text, numeric_value, unit, trend_text)
VALUES
  ('60000000-0000-4000-8000-000000000001', 'cpu', 'CPU', '32%', 32, '%', '+4%'),
  ('60000000-0000-4000-8000-000000000002', 'memory', '内存', '58%', 58, '%', '-2%'),
  ('60000000-0000-4000-8000-000000000003', 'upload', '上传', '18 MB/s', 18, 'MB/s', '稳定'),
  ('60000000-0000-4000-8000-000000000004', 'download', '下载', '42 MB/s', 42, 'MB/s', '+12%')
ON CONFLICT (id) DO UPDATE SET value_text = EXCLUDED.value_text, numeric_value = EXCLUDED.numeric_value, trend_text = EXCLUDED.trend_text, observed_at = now();

INSERT INTO alerts (id, title, detail, tone, source, status, metadata)
VALUES
  ('61000000-0000-4000-8000-000000000001', '备份中', 'MacBook Pro 文档备份 72%', 'blue', 'backup-sync', 'running', '{"icon": "ArchiveRestore"}'),
  ('61000000-0000-4000-8000-000000000002', '权限审计', '3 个分享链接建议收紧', 'orange', 'security-center', 'queued', '{"icon": "ShieldCheck"}'),
  ('61000000-0000-4000-8000-000000000003', '本地模型', '隐私空间强制本地推理', 'green', 'ai-assistant', 'succeeded', '{"icon": "Bot"}')
ON CONFLICT (id) DO UPDATE SET detail = EXCLUDED.detail, tone = EXCLUDED.tone, status = EXCLUDED.status, metadata = EXCLUDED.metadata;

INSERT INTO backup_plans (id, name, source_uri, target_uri, schedule_cron, enabled)
VALUES
  ('62000000-0000-4000-8000-000000000001', 'MacBook Pro 文档备份', 'file:///Users/demo/Documents', 'higoos://backup-archive/macbook-pro', '15 */2 * * *', true)
ON CONFLICT (name) DO UPDATE SET source_uri = EXCLUDED.source_uri, target_uri = EXCLUDED.target_uri, schedule_cron = EXCLUDED.schedule_cron;

INSERT INTO backup_runs (id, plan_id, status, progress)
VALUES
  ('63000000-0000-4000-8000-000000000001', '62000000-0000-4000-8000-000000000001', 'running', 72)
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, progress = EXCLUDED.progress;

INSERT INTO download_tasks (id, name, task_kind, status, target_path, size_bytes, progress, created_by)
VALUES
  ('70000000-0000-4000-8000-000000000001', '未归档发票批量导入', 'http', 'running', '/财务票据/下载目录/未归档发票', 642000000, 72, '00000000-0000-4000-8000-000000000001'),
  ('70000000-0000-4000-8000-000000000002', '五一旅行素材归档', 'magnet', 'queued', '/照片与视频/五一旅行照片精选', 4200000000, 0, '00000000-0000-4000-8000-000000000001')
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, progress = EXCLUDED.progress;

INSERT INTO download_sources (id, task_id, uri, priority, metadata)
VALUES
  ('71000000-0000-4000-8000-000000000001', '70000000-0000-4000-8000-000000000001', 'fixture://nas-root/downloads/unarchived-invoices.txt', 10, '{"source": "fixture"}')
ON CONFLICT (id) DO UPDATE SET uri = EXCLUDED.uri, priority = EXCLUDED.priority;

INSERT INTO archive_rules (id, name, source_path, target_path, matcher)
VALUES
  ('72000000-0000-4000-8000-000000000001', '发票按年月归档', '/下载/票据', '/财务票据/{yyyy}/{mm}', '{"extensions": ["pdf", "jpg", "png"], "ocr_keywords": ["发票", "税额"]}')
ON CONFLICT (name) DO UPDATE SET source_path = EXCLUDED.source_path, target_path = EXCLUDED.target_path, matcher = EXCLUDED.matcher;

INSERT INTO media_items (id, file_id, media_type, title, taken_at, metadata)
VALUES
  ('80000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000103', 'photo', '五一旅行照片精选', now() - interval '5 days', '{"photos": 832, "selected": 124, "videos": 18}')
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, metadata = EXCLUDED.metadata;

INSERT INTO albums (id, name, description, cover_media_id)
VALUES
  ('81000000-0000-4000-8000-000000000001', '五一旅行照片精选', '832 张照片，已筛出 124 张清晰照片和 18 段视频。', '80000000-0000-4000-8000-000000000001')
ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description, cover_media_id = EXCLUDED.cover_media_id;

INSERT INTO memory_runs (id, name, status, input_filter, output_uri)
VALUES
  ('82000000-0000-4000-8000-000000000001', '五一旅行回忆生成', 'running', '{"album": "五一旅行照片精选"}', 'fixture://nas-root/photos-and-media/mayday-memory.md')
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, input_filter = EXCLUDED.input_filter, output_uri = EXCLUDED.output_uri;

INSERT INTO compose_stacks (id, name, compose_path, status)
VALUES
  ('90000000-0000-4000-8000-000000000001', 'media-stack', '/docker/media-stack/compose.yaml', 'running'),
  ('90000000-0000-4000-8000-000000000002', 'download-stack', '/docker/download-stack/compose.yaml', 'running')
ON CONFLICT (name) DO UPDATE SET compose_path = EXCLUDED.compose_path, status = EXCLUDED.status;

INSERT INTO containers (id, stack_id, container_name, image, state, ports, cpu_percent, memory_bytes)
VALUES
  ('91000000-0000-4000-8000-000000000001', '90000000-0000-4000-8000-000000000001', 'higo-photos', 'ghcr.io/higoos/photos:dev', 'running', '[{"host": 2283, "container": 3001}]', 5.4, 512000000),
  ('91000000-0000-4000-8000-000000000002', '90000000-0000-4000-8000-000000000001', 'higo-transcoder', 'ghcr.io/higoos/transcoder:dev', 'running', '[]', 12.1, 768000000),
  ('91000000-0000-4000-8000-000000000003', '90000000-0000-4000-8000-000000000002', 'higo-downloader', 'ghcr.io/higoos/downloader:dev', 'running', '[{"host": 6881, "container": 6881}]', 8.8, 384000000),
  ('91000000-0000-4000-8000-000000000004', '90000000-0000-4000-8000-000000000002', 'higo-rss', 'ghcr.io/higoos/rss:dev', 'running', '[]', 1.6, 128000000)
ON CONFLICT (container_name) DO UPDATE SET image = EXCLUDED.image, state = EXCLUDED.state, ports = EXCLUDED.ports, cpu_percent = EXCLUDED.cpu_percent, memory_bytes = EXCLUDED.memory_bytes;

INSERT INTO app_catalog (id, app_key, name, category, image, default_config)
VALUES
  ('92000000-0000-4000-8000-000000000001', 'photo-media', '相册媒体', 'media', 'ghcr.io/higoos/photos:dev', '{"port": 2283}'),
  ('92000000-0000-4000-8000-000000000002', 'download-center', '下载中心', 'download', 'ghcr.io/higoos/downloader:dev', '{"bt_port": 6881}')
ON CONFLICT (app_key) DO UPDATE SET name = EXCLUDED.name, category = EXCLUDED.category, image = EXCLUDED.image, default_config = EXCLUDED.default_config;

INSERT INTO remote_channels (id, name, channel_type, status, config)
VALUES
  ('93000000-0000-4000-8000-000000000001', '家庭远程访问', 'tunnel', 'succeeded', '{"mfa": true, "share_scan": true}'),
  ('93000000-0000-4000-8000-000000000002', 'higoos-dev-ddns', 'ddns', 'succeeded', '{"hostname": "dev.higoos.local"}')
ON CONFLICT (name) DO UPDATE SET status = EXCLUDED.status, config = EXCLUDED.config;

INSERT INTO ddns_records (id, hostname, provider, last_ip, status)
VALUES
  ('94000000-0000-4000-8000-000000000001', 'dev.higoos.local', 'devstub', '127.0.0.1', 'succeeded')
ON CONFLICT (hostname) DO UPDATE SET provider = EXCLUDED.provider, last_ip = EXCLUDED.last_ip, status = EXCLUDED.status;

INSERT INTO agent_templates (id, template_key, name, description, tools, risk_level)
VALUES
  ('a0000000-0000-4000-8000-000000000001', 'family-docs-assistant', '家庭资料助手', '整理保修单、说明书、证件和医疗资料，提供问答与提醒。', '["文件搜索", "摘要", "提醒", "分享"]', 'medium'),
  ('a0000000-0000-4000-8000-000000000002', 'project-docs-agent', '项目资料 Agent', '汇总项目文件、合同、会议纪要和素材，生成资料包。', '["语义搜索", "文件夹摘要", "打包", "权限检查"]', 'medium'),
  ('a0000000-0000-4000-8000-000000000003', 'ops-agent', '设备运维 Agent', '监控硬盘、备份、Docker 和网络状态，异常时建议处理。', '["设备监控", "备份检查", "通知", "日志读取"]', 'low')
ON CONFLICT (template_key) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, tools = EXCLUDED.tools, risk_level = EXCLUDED.risk_level;

INSERT INTO workflow_definitions (id, agent_template_id, workflow_key, name, definition, version)
VALUES
  ('a1000000-0000-4000-8000-000000000001', 'a0000000-0000-4000-8000-000000000001', 'archive-invoices-review', '下载目录智能整理', '{"nodes": [{"label": "触发", "value": "新文件进入下载目录"}, {"label": "理解", "value": "OCR + 发票识别 + 重复检测"}, {"label": "确认", "value": "中风险，等待用户确认"}, {"label": "执行", "value": "重命名、归档、写入审计"}]}', 1)
ON CONFLICT (workflow_key) DO UPDATE SET definition = EXCLUDED.definition, version = EXCLUDED.version;

INSERT INTO workflow_runs (id, workflow_id, status, input, started_by)
VALUES
  ('a2000000-0000-4000-8000-000000000001', 'a1000000-0000-4000-8000-000000000001', 'queued', '{"path": "/下载/票据"}', '00000000-0000-4000-8000-000000000001')
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, input = EXCLUDED.input;

INSERT INTO confirmations (id, workflow_run_id, requested_by, status, title, detail)
VALUES
  ('a3000000-0000-4000-8000-000000000001', 'a2000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000001', 'queued', '下载目录智能整理', '中风险，等待用户确认后执行重命名、归档、写入审计。')
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, detail = EXCLUDED.detail;

INSERT INTO conversation_threads (id, user_id, title, metadata)
VALUES
  ('b0000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000001', '客户 A 合同与备份确认', '{"source": "web-pc demo"}')
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, metadata = EXCLUDED.metadata, updated_at = now();

INSERT INTO messages (id, thread_id, role, content, model_id, metadata, created_at)
VALUES
  ('b1000000-0000-4000-8000-000000000001', 'b0000000-0000-4000-8000-000000000001', 'user', '找一下上个月客户 A 的最终合同，并确认有没有备份。', NULL, '{}', now() - interval '3 minutes'),
  ('b1000000-0000-4000-8000-000000000002', 'b0000000-0000-4000-8000-000000000001', 'assistant', '找到了 1 份最终版合同，位于团队空间/客户A/合同。该文件已进入每日快照和异地备份，权限为项目组可见。', '40000000-0000-4000-8000-000000000002', '{"citations": 1}', now() - interval '2 minutes'),
  ('b1000000-0000-4000-8000-000000000003', 'b0000000-0000-4000-8000-000000000001', 'assistant', '我还发现 3 个相关附件未加入项目资料图谱，是否需要生成整理计划？', '40000000-0000-4000-8000-000000000002', '{"action": "generate_plan"}', now() - interval '1 minute')
ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, metadata = EXCLUDED.metadata;

INSERT INTO retrieval_citations (id, message_id, file_id, chunk_id, label, excerpt, score)
VALUES
  ('b2000000-0000-4000-8000-000000000001', 'b1000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000102', '44000000-0000-4000-8000-000000000002', '团队空间/客户A/合同', '最终版合同已进入每日快照和异地备份。', 0.97000)
ON CONFLICT (id) DO UPDATE SET excerpt = EXCLUDED.excerpt, score = EXCLUDED.score;

INSERT INTO assistant_actions (id, thread_id, message_id, action_key, status, payload)
VALUES
  ('b3000000-0000-4000-8000-000000000001', 'b0000000-0000-4000-8000-000000000001', 'b1000000-0000-4000-8000-000000000003', 'generate_project_graph_plan', 'queued', '{"related_attachments": 3}')
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, payload = EXCLUDED.payload;

INSERT INTO settings (key, value, category, updated_by)
VALUES
  ('network.remote_access.enabled', 'true', 'remote', '00000000-0000-4000-8000-000000000001'),
  ('ai.private_space.runtime', '"local"', 'ai', '00000000-0000-4000-8000-000000000001'),
  ('downloads.auto_archive.enabled', 'true', 'downloads', '00000000-0000-4000-8000-000000000001')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, category = EXCLUDED.category, updated_by = EXCLUDED.updated_by, updated_at = now();

COMMIT;
