import React from 'react';
import './Card.css';

interface CardSubComponentProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
}

export const CardHeader: React.FC<CardSubComponentProps> = ({ className = '', children, ...props }) => (
  <div className={`card-header ${className}`} {...props}>{children}</div>
);

export const CardTitle: React.FC<React.HTMLAttributes<HTMLHeadingElement>> = ({ className = '', children, ...props }) => (
  <h3 className={`card-title ${className}`} {...props}>{children}</h3>
);

export const CardDescription: React.FC<React.HTMLAttributes<HTMLParagraphElement>> = ({ className = '', children, ...props }) => (
  <p className={`card-description ${className}`} {...props}>{children}</p>
);

export const CardContent: React.FC<CardSubComponentProps> = ({ className = '', children, ...props }) => (
  <div className={`card-content ${className}`} {...props}>{children}</div>
);

export const CardFooter: React.FC<CardSubComponentProps> = ({ className = '', children, ...props }) => (
  <div className={`card-footer ${className}`} {...props}>{children}</div>
);

interface CardProps extends Omit<React.HTMLAttributes<HTMLDivElement>, 'title'> {
  title?: React.ReactNode;
  description?: React.ReactNode;
  footer?: React.ReactNode;
  headerAction?: React.ReactNode;
}

export const Card: React.FC<CardProps> = ({
  title,
  description,
  footer,
  headerAction,
  children,
  className = '',
  ...props
}) => {
  return (
    <div className={`card ${className}`} {...props}>
      {(title || description || headerAction) && (
        <CardHeader>
          <div>
            {title && <CardTitle>{title}</CardTitle>}
            {description && <CardDescription>{description}</CardDescription>}
          </div>
          {headerAction && <div className="card-action">{headerAction}</div>}
        </CardHeader>
      )}
      <CardContent>{children}</CardContent>
      {footer && <CardFooter>{footer}</CardFooter>}
    </div>
  );
};
