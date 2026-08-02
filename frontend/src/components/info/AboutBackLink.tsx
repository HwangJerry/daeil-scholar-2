// AboutBackLink — Direct return action from foundation detail pages to the about hub
import { ArrowLeft } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button } from '../ui/Button';

export function AboutBackLink() {
  return (
    <Button asChild variant="outline" size="sm">
      <Link to="/about">
        <ArrowLeft className="h-4 w-4" aria-hidden />
        장학회 소개로 돌아가기
      </Link>
    </Button>
  );
}
