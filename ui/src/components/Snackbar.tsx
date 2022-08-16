import Slide from '@mui/material/Slide';
import Snackbar from '@mui/material/Snackbar/Snackbar';

function TransitionLeft(props) {
  return <Slide {...props} direction="left" />;
}

export default function VackupSnackbar() {
  return <Snackbar TransitionComponent={TransitionLeft} />;
}