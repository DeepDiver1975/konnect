import React from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';

import { withStyles } from '@material-ui/core/styles';
import Grid from '@material-ui/core/Grid';
import DialogContent from '@material-ui/core/DialogContent';
import DialogActions from '@material-ui/core/DialogActions';
import LocaleSelect from 'kpop/es/IntlContainer/LocaleSelect';

import ResponsiveDialog from './ResponsiveDialog';
import KopanoLogo from '../images/kopano-logo.svg';
import Loading from './Loading';

const styles = theme => ({
  root: {
    display: 'flex',
    flex: 1
  },
  content: {
    paddingTop: 24,
    paddingBottom: 12,
    minHeight: 500,
    paddingLeft: 2 * theme.spacing.unit,
    paddingRight: 2 * theme.spacing.unit,
    position: 'relative'
  },
  logo: {
    height: 18,
    marginBottom: theme.spacing.unit * 2
  },
  actions: {
    marginTop: -40,
    justifyContent: 'flex-start',
    paddingLeft: 3 * theme.spacing.unit,
    paddingRight: 3 * theme.spacing.unit
  }
});

function ResponsiveScreen(props) {
  const {
    classes,
    withoutLogo,
    withoutPadding,
    loading,
    children,
    className,
    DialogProps,
    PaperProps,
    ...other
  } = props;

  const logo = withoutLogo ? null :
    <DialogContent><img src={KopanoLogo} className={classes.logo} alt="Kopano"/></DialogContent>;

  const content = loading ? <Loading/> : (withoutPadding ? children : <DialogContent>{children}</DialogContent>);

  return (
    <Grid container justify="center" alignItems="center" spacing={0}
      className={classNames(classes.root, className)} {...other}>
      <ResponsiveDialog open fullWidth maxWidth="sm"
        disableBackdropClick disableEscapeKeyDown hideBackdrop
        {...DialogProps}
        PaperProps={{elevation: 4, ...PaperProps}}
      >
        <div className={classes.content}>
          {logo}
          {content}
        </div>
        <DialogActions className={classes.actions} disableActionSpacing><LocaleSelect disableUnderline/></DialogActions>
      </ResponsiveDialog>
    </Grid>
  );
}

ResponsiveScreen.defaultProps = {
  withoutLogo: false,
  withoutPadding: false,
  loading: false
};

ResponsiveScreen.propTypes = {
  classes: PropTypes.object.isRequired,
  withoutLogo: PropTypes.bool,
  withoutPadding: PropTypes.bool,
  children: PropTypes.node.isRequired,
  className: PropTypes.string,
  PaperProps: PropTypes.object,
  DialogProps: PropTypes.object
};

export default withStyles(styles)(ResponsiveScreen);
