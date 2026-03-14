import React from 'react';
import { FiChevronDown } from 'react-icons/fi';
import './Select.css';

interface SelectOption {
  label: string;
  value: string | number;
  disabled?: boolean;
}

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
  options: SelectOption[];
  layout?: 'vertical' | 'horizontal';
}

export const Select: React.FC<SelectProps> = ({
  label,
  error,
  options,
  className = '',
  layout = 'vertical',
  id,
  ...props
}) => {
  const generatedId = React.useId();
  const selectId = id || props.name || generatedId;

  return (
    <div className={`select-wrapper ${layout} ${className}`}>
      {label && (
        <label htmlFor={selectId} className="input-label">
          {label}
        </label>
      )}
      <div className="select-container">
        <select
          id={selectId}
          className={`select-field ${error ? 'has-error' : ''}`}
          {...props}
        >
          {options.map((option) => (
            <option 
              key={option.value} 
              value={option.value} 
              disabled={option.disabled}
            >
              {option.label}
            </option>
          ))}
        </select>
        <span className="select-arrow">
          <FiChevronDown />
        </span>
      </div>
      {error && <span className="select-error">{error}</span>}
    </div>
  );
};
