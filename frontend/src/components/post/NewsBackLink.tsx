// NewsBackLink — Direct return action from a news detail page to the news list
import { ArrowLeft } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button } from '../ui/Button';

export function NewsBackLink() {
  return (
    <Button asChild variant="outline" size="sm">
      <Link to="/">
        <ArrowLeft className="h-4 w-4" aria-hidden />
        소식 목록으로 돌아가기
      </Link>
    </Button>
  );
}
