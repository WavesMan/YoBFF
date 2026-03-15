import React from 'react';
import './Textarea.css';

interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
  layout?: 'vertical' | 'horizontal';
}

/**
 *
 * 文本域组件，用于输入较长内容。
 *
 */
export const Textarea: React.FC<TextareaProps> = ({
  label,
  error,
  className = '',
  layout = 'vertical',
  id,
  ...props
}) => {
  const generatedId = React.useId();
  const inputId = id || props.name || generatedId;
  
  return (
    <div className={`input-wrapper ${layout} ${className}`}>
      {label && (
        <label htmlFor={inputId} className="input-label">
          {label}
        </label>
      )}
      <div className="input-container">
        <textarea
          id={inputId}
          className={`input-field textarea-field ${error ? 'has-error' : ''}`}
          {...props}
        />
      </div>
      {error && <span className="input-error">{error}</span>}
    </div>
  );
};
