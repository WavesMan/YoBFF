import { useState, useEffect } from 'react';
import { FiX, FiAlertCircle, FiList, FiTrash2, FiCopy, FiCheck } from 'react-icons/fi';
import { translateMessage, type ErrorLogItem } from './ErrorDrawerUtils';
import './ErrorDrawer.css';

interface ErrorDrawerProps {
    errors: ErrorLogItem[];
    onClear: () => void;
}

export function ErrorDrawer({ errors, onClear }: ErrorDrawerProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [isHovered, setIsHovered] = useState(false);
    const [copiedId, setCopiedId] = useState<string | null>(null);

    // 当有新错误时，可能需要提示一下（可选）
    useEffect(() => {
        if (errors.length > 0 && !isOpen) {
            // 可以加一个动画效果
        }
    }, [errors.length, isOpen]);

    const handleCopy = async (text: string, id: string) => {
        try {
            await navigator.clipboard.writeText(text);
            setCopiedId(id);
            setTimeout(() => setCopiedId(null), 2000);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    };

    const handleCopyAll = async () => {
        try {
            const allText = errors.map(e => {
                const time = new Date(e.timestamp).toLocaleString();
                const code = e.errorCode ? `[${e.errorCode}]` : '';
                const msg = translateMessage(e.message, e.errorCode);
                const reqId = e.requestId ? `[request_id=${e.requestId}]` : '';
                return `${time} ${code} ${msg} ${reqId}`;
            }).join('\n');
            await navigator.clipboard.writeText(allText);
            setCopiedId('all');
            setTimeout(() => setCopiedId(null), 2000);
        } catch (err) {
            console.error('Failed to copy all:', err);
        }
    };

    if (errors.length === 0) return null;

    return (
        <>
            {/* 侧边悬浮按钮 - 始终显示在右侧边缘 */}
            <div 
                className={`error-drawer-trigger ${isOpen ? 'hidden' : ''}`}
                onClick={() => setIsOpen(true)}
                onMouseEnter={() => setIsHovered(true)}
                onMouseLeave={() => setIsHovered(false)}
                title="点击查看错误日志"
                style={{ right: isOpen ? '-100px' : '0' }}
            >
                <FiAlertCircle size={20} />
                {isHovered && <span>错误日志</span>}
                <span className="error-drawer-count">{errors.length}</span>
            </div>

            {/* 抽屉面板 */}
            <div className={`error-drawer-panel ${isOpen ? 'open' : ''}`}>
                <div className="error-drawer-header">
                    <div className="error-drawer-title">
                        <FiList className="text-error" />
                        <span>错误日志 ({errors.length})</span>
                    </div>
                    <div className="error-drawer-actions">
                         <button 
                            className="error-drawer-action-btn"
                            onClick={handleCopyAll}
                            title="复制全部"
                        >
                            {copiedId === 'all' ? <FiCheck size={16} /> : <FiCopy size={16} />}
                        </button>
                        <button 
                            className="error-drawer-action-btn" 
                            onClick={onClear}
                            title="清空日志"
                        >
                            <FiTrash2 size={16} />
                        </button>
                        <button 
                            className="error-drawer-close-btn" 
                            onClick={() => setIsOpen(false)}
                            title="关闭"
                        >
                            <FiX size={20} />
                        </button>
                    </div>
                </div>

                <div className="error-drawer-list">
                    {errors.map((error) => (
                        <div key={error.id} className="error-item">
                            <div className="error-item-header">
                                <span>{new Date(error.timestamp).toLocaleTimeString()}</span>
                                {error.errorCode && (
                                    <span className="error-code-badge">[{error.errorCode}]</span>
                                )}
                                <button 
                                    className="error-item-copy-btn"
                                    onClick={() => {
                                        const text = `[${error.errorCode || 'error'}] ${translateMessage(error.message, error.errorCode)} ${error.requestId ? `[request_id=${error.requestId}]` : ''}`;
                                        handleCopy(text, error.id);
                                    }}
                                    title="复制此条信息"
                                >
                                    {copiedId === error.id ? <FiCheck size={14} /> : <FiCopy size={14} />}
                                </button>
                            </div>
                            <div className="error-item-content">
                                {translateMessage(error.message, error.errorCode)}
                            </div>
                            {error.requestId && (
                                <code className="error-request-id">
                                    [request_id={error.requestId}]
                                </code>
                            )}
                        </div>
                    ))}
                    {errors.length === 0 && (
                        <div className="error-empty-state">
                            暂无错误记录
                        </div>
                    )}
                </div>
            </div>
            
            {/* 遮罩层 - 点击外部关闭 */}
            {isOpen && (
                <div 
                    className="error-drawer-overlay"
                    onClick={() => setIsOpen(false)}
                />
            )}
        </>
    );
}
