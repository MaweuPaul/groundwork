import React from 'react';
import Border from './border';

interface CardProps {
  title: string;
  children: React.ReactNode;
}

export const Card = ({ title, children }: CardProps) => {
  return (
    <Border>
      <div className="flex  shadow-lg flex-col gap-2 bg-white">
        <h3 className="text-lg font-semibold">{title}</h3>
        {children}
      </div>
    </Border>
  );
};
