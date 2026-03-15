import React, { useRef, useState } from 'react';
import Editor, { type OnMount } from '@monaco-editor/react';
import { FiMaximize2 } from 'react-icons/fi';
import { Modal } from './Modal';
import './CodeEditor.css';

interface CodeEditorProps {
  value: string;
  onChange?: (value: string | undefined) => void;
  language?: string;
  readOnly?: boolean;
  height?: string | number;
  label?: string;
  className?: string;
}

export const CodeEditor: React.FC<CodeEditorProps> = ({
  value,
  onChange,
  language = 'json',
  readOnly = false,
  height = '300px',
  label,
  className,
}) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);

  const handleEditorDidMount: OnMount = (editor) => {
    editorRef.current = editor;
  };

  const openModal = () => setIsModalOpen(true);
  const closeModal = () => setIsModalOpen(false);

  return (
    <div className={`code-editor-container ${className || ''}`}>
      {label && (
        <div className="code-editor-header">
          <span className="code-editor-label">{label}</span>
          <button 
            type="button" 
            className="code-editor-expand-btn" 
            onClick={openModal}
            title="全屏编辑"
          >
            <FiMaximize2 size={14} />
          </button>
        </div>
      )}
      
      <div className="code-editor-wrapper" style={{ height: typeof height === 'number' ? `${height}px` : height }}>
        <Editor
          height="100%"
          defaultLanguage={language}
          language={language}
          value={value}
          onChange={onChange}
          theme="vs-dark"
          options={{
            readOnly,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            fontSize: 13,
            automaticLayout: true,
          }}
          onMount={handleEditorDidMount}
        />
      </div>

      <Modal
        isOpen={isModalOpen}
        onClose={closeModal}
        title={label || '编辑代码'}
        width="80vw"
        bodyClassName="code-editor-modal-body"
      >
        <div style={{ height: '70vh' }}>
          <Editor
            height="100%"
            defaultLanguage={language}
            language={language}
            value={value}
            onChange={onChange}
            theme="vs-dark"
            options={{
              readOnly,
              minimap: { enabled: true },
              scrollBeyondLastLine: false,
              fontSize: 14,
              automaticLayout: true,
            }}
          />
        </div>
      </Modal>
    </div>
  );
};
